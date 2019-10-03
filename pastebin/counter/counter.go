package counter

import (
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
)

var pool *redis.Pool

func InitRedisPool(addr string) {
	pool = newPool(addr)
}

func newPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 300 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr)
		},
	}
}

/*
// Last retrieves the latest timestamp value across all counter shards for a given name
func Last(name string) (time.Time, error) {

}
*/

// Increment increments the named counter and returns the count
func Count(name string) int {
	var count int
	conn := pool.Get()

	ount, err := conn.Do("HINCRBY", name, "count", 1)
	if err != nil {
		log.Printf("ERROR: Increment operation failed for %s, %v", name, err)
	}

	count, err = redis.Int(ount, err)
	if err != nil {
		log.Printf("ERROR: Increment operation failed for %s, %v", name, err)
	}

	_, err = conn.Do("HSET", name, "last", time.Now().Format("2006-02-01"))
	if err != nil {
		log.Printf("ERROR: Date operation failed for %s, %v", name, err)
	}

	err = conn.Close()
	if err != nil {
		log.Printf("ERROR: Increment closure failed for %s, %v", name, err)
	}

	return count
}

func Delete(name string) {
	conn := pool.Get()

	_, err := conn.Do("DEL", name)
	if err != nil {
		log.Printf("ERROR: Date operation failed for %s, %v", name, err)
	}

	err = conn.Close()
	if err != nil {
		log.Printf("ERROR: Delete closure failed for %s, %v", name, err)
	}
}
