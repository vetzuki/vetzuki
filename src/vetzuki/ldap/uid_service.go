package ldap

import (
	"fmt"
	"github.com/go-redis/redis"
	"log"
	"os"
	"strconv"
)

const (
	baseUID = 10000
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

// NextUID : Provide the next UID number
func NextUID() (int64, error) {
	log.Printf("debug: connecting to redis://%s@%s/%d", redisPassword, redisHost, redisDB)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost,     // use default Addr
		Password: redisPassword, // no password set
		DB:       redisDB,       // use default DB
	})
	if rdb == nil {
		log.Printf("error: unable to connect to redis")
		return int64(0), fmt.Errorf("unable to connect to redis")
	}

	uid, err := rdb.Incr("nextUID").Result()
	if err != nil {
		log.Printf("error: unable to create next key: %s", err)
	}
	return uid + baseUID, err
}
