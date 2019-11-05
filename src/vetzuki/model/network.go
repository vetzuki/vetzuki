package model

import (
	"fmt"
	"github.com/go-redis/redis"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
)

const (
	baseSSHPort        = 10000
	defaultNetworkMask = "255.255.255.0"
	maxNetworks        = 65534
	sshPortField       = "SSHPort"
)

var (
	redisHost        = "localhost:6379"
	redisPassword    = ""
	redisDB          = 0
	envRedisHost     = "REDIS_HOST"
	envRedisPassword = "REDIS_PASSWORD"
	envRedisDB       = "REDIS_DB"
)

func init() {
	if b := os.Getenv(envRedisHost); len(b) > 0 {
		redisHost = b
	}
	if b := os.Getenv(envRedisPassword); len(b) > 0 {
		redisPassword = b
	}
	if b := os.Getenv(envRedisDB); len(b) > 0 {
		if v, err := strconv.Atoi(b); err == nil {
			redisDB = v
		} else {
			log.Printf("error: unable to convert %s to int: %s", b, err)
		}
	}
}

// Network : Information required to create a network on an ec2 instance
type Network struct {
	ID                 int       `sql:"id" json:"id"`
	EC2InstanceID      string    `sql:"ec2_instance_id" json:"ec2InstanceID"`
	Network            string    `sql:"network" json:"network"`
	Mask               string    `sql:"mask" json:"mask"`
	ProspectID         int64     `sql:"prospect_id" json:"prospectID"`
	ExamContainerIP    string    `sql:"exam_container_ip" json:"examContainerIP"`
	ProctorContainerIP string    `sql:"proctor_container_ip" json:"proctorContainerIP"`
	GatewayIP          string    `sql:"gateway_ip" json:"gatewayIP"`
	SSHPort            int       `sql:"ssh_port" json:"SSHPort"`
	Created            time.Time `sql:"created" json:"created"`
	Modified           time.Time `sql:"modified" json:"modified"`
}

// CreateProspectNetwork : Create a network in the database
func CreateProspectNetwork(prospectID int64, ec2InstanceID string) (*Network, bool) {
	base10Bits, ok := NextNetwork(ec2InstanceID)
	if !ok {
		log.Printf("error: failed to get next network")
		return nil, false
	}
	sshPort, ok := NextPort(ec2InstanceID)
	if !ok {
		log.Printf("error: failed to get next SSH port")
		return nil, false
	}
	ipv4Network := calculateNetwork(base10Bits, 0)
	rand.Seed(time.Now().Unix())
	network := &Network{
		EC2InstanceID:   ec2InstanceID,
		Network:         ipv4Network,
		Mask:            defaultNetworkMask,
		ProspectID:      prospectID,
		ExamContainerIP: calculateNetwork(base10Bits, 2),
		GatewayIP:       calculateNetwork(base10Bits, 1),
		ProctorContainerIP: calculateNetwork(
			base10Bits,
			rand.Int63n(24)+230),
		SSHPort: sshPort,
	}
	err := connection.QueryRow(`
	INSERT INTO prospect_network
	(ec2_instance_id, network, mask, prospect_id, exam_container_ip, proctor_container_ip, gateway_ip, ssh_port)
	VALUES
	($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, created, modified
	`,
		network.EC2InstanceID,
		network.Network,
		network.Mask,
		network.ProspectID,
		network.ExamContainerIP,
		network.ProctorContainerIP,
		network.GatewayIP,
		network.SSHPort,
	).Scan(&network.ID, &network.Created, &network.Modified)
	if err != nil {
		log.Printf("error: failed to create network for %d on %s: %s", prospectID, ec2InstanceID, err)
		return nil, false
	}
	return network, true
}

func redisConnection() *redis.Client {
	log.Printf("debug: connecting to redis://%s@%s/%d", redisPassword, redisHost, redisDB)
	return redis.NewClient(&redis.Options{
		Addr:     redisHost,     // use default Addr
		Password: redisPassword, // no password set
		DB:       redisDB,       // use default DB
	})
}

// NextNetwork : Reserve the next free network on the EC2 Instance.
func NextNetwork(ec2InstanceID string) (int64, bool) {
	log.Printf("debug: getting next network for %s", ec2InstanceID)
	c := redisConnection()
	if c == nil {
		log.Printf("error: unable to get Redis connection")
		return 0, false
	}
	if e, err := c.HExists(ec2InstanceID, "network").Result(); err != nil {
		log.Printf("error: while finding network field for %s: %s", ec2InstanceID, err)
		return 0, false
	} else if !e {
		wasAdded, setErr := c.HSet(ec2InstanceID, "network", 0).Result()
		if setErr != nil {
			log.Printf("error: unable to set network key for %s to 0: %s", ec2InstanceID, err)
			return 0, false
		}
		if wasAdded {
			log.Printf("debug: creating first network for %s", ec2InstanceID)
			return 0, true
		}
	}
	nextNetwork, err := c.HIncrBy(ec2InstanceID, "network", 1).Result()
	if err != nil {
		log.Printf("error: failed to get next network for %s: %s", ec2InstanceID, err)
		return 0, false
	}
	return nextNetwork, true
}

// DeleteProspectNetwork : Delete a prospect network
func DeleteProspectNetwork(id int) bool {
	log.Printf("debug: deleting network %d", id)
	row := connection.QueryRow(`delete from prospect_network where id = $1`, id)
	return row != nil
}

// FindProspectNetwork : Find a network by prospectID
func FindProspectNetwork(prospectID int) (*Network, bool) {
	log.Printf("debug: finding network for %d", prospectID)
	var network Network
	err := connection.QueryRow(
		`SELECT id, ec2_instance_id, network, mask,
	  prospect_id, exam_container_ip, proctor_container_ip, gateway_ip, ssh_port,
	  created, modified
	FROM prospect_network
	WHERE prospect_id = $1`, prospectID,
	).Scan(
		&network.ID,
		&network.EC2InstanceID,
		&network.Mask,
		&network.ProspectID,
		&network.ExamContainerIP,
		&network.ProctorContainerIP,
		&network.GatewayIP,
		&network.SSHPort,
		&network.Created,
		&network.Modified,
	)
	if err != nil {
		log.Printf("error: failed to find prospect network for %d: %s", prospectID, err)
		return nil, false
	}
	return &network, true
}

// ReleaseNetwork : The encoding used for networks doesn't permit a release. This
// adds unused networks to a list of free networks.
func ReleaseNetwork(ec2InstanceID, network string) bool {
	log.Printf("debug: releasing %s from %s", network, ec2InstanceID)
	id := fmt.Sprintf("%s_%s", ec2InstanceID, network)
	row := connection.QueryRow(`
	INSERT INTO network_list
	(id, ec2_instance_id)
	VALUES ($1, $2)
	`, id, ec2InstanceID)
	if row == nil {
		log.Printf("warning: failed to create network list entry")
		return false
	}
	return true
}

func calculateNetwork(i, host int64) string {
	log.Printf("debug: calculating network for %d with host %d", i, host)
	if i >= maxNetworks {
		log.Printf("warning: %d is greater than %d, network overlap may occur", i, maxNetworks)
	}
	return net.IPv4(
		byte(10),
		byte(i>>8),
		byte(i),
		byte(host)).String()
}

// NextPort : Reserve the next free SSH port on the EC2 Instance.
func NextPort(ec2InstanceID string) (int, bool) {
	c := redisConnection()
	if c == nil {
		log.Printf("error: unable to get redis connection")
		return 0, false
	}
	exists, err := c.HExists(ec2InstanceID, sshPortField).Result()
	if err != nil {
		log.Printf("error: while checking ssh ports field for %s: %s", ec2InstanceID, err)
		return 0, false
	}
	sshPort := baseSSHPort
	if !exists {
		if b, err := c.HSet(ec2InstanceID, sshPortField, sshPort).Result(); err != nil {
			log.Printf("error: while setting ssh ports field for %s: %s", ec2InstanceID, err)
			return 0, false
		} else if !b {
			log.Printf("warning: setting initial ssh port field for %s failed, retrying", ec2InstanceID)
			return NextPort(ec2InstanceID)
		}
		return sshPort, true
	}
	p, err := c.HIncrBy(ec2InstanceID, sshPortField, 1).Result()
	if err != nil {
		log.Printf("error: unable to get next ssh port for %s: %s", ec2InstanceID, err)
		return 0, false
	}
	// p will always be in the range of int()
	return int(p), true
}
