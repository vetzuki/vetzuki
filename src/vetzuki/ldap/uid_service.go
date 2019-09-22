package ldap

import (
	"github.com/go-redis/redis"
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
		}
	}
}

// NextUID : Provide the next UID number
func NextUID() (int64, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisHost,     // use default Addr
		Password: redisPassword, // no password set
		DB:       redisDB,       // use default DB
	})

	uid, err := rdb.Incr("nextUID").Result()
	return uid + baseUID, err
}
