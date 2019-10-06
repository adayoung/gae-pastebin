package counter

import (
	"log"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/adayoung/gae-pastebin/pastebin/utils"
)

// Increment increments the named counter and returns the count
func Count(name string) int {
	var count int
	var err error
	conn := utils.RedisPool.Get()

	if ount, err := conn.Do("HINCRBY", name, "count", 1); err != nil {
		log.Printf("ERROR: Increment operation failed for %s, %v", name, err)
	} else if count, err = redis.Int(ount, err); err != nil {
		log.Printf("ERROR: Increment operation failed for %s, %v", name, err)
	} else if _, err = conn.Do("HSET", name, "last", time.Now().Format("2006-02-01")); err != nil {
		log.Printf("ERROR: Date operation failed for %s, %v", name, err)
	}

	err = conn.Close()
	if err != nil {
		log.Printf("ERROR: Increment closure failed for %s, %v", name, err)
	}

	return count
}

func Delete(name string) {
	conn := utils.RedisPool.Get()

	_, err := conn.Do("DEL", name)
	if err != nil {
		log.Printf("ERROR: Delete operation failed for %s, %v", name, err)
	}

	err = conn.Close()
	if err != nil {
		log.Printf("ERROR: Delete closure failed for %s, %v", name, err)
	}
}

/*
// Last retrieves the latest timestamp value across all counter shards for a given name
func Last(name string) (time.Time, error) {

}
*/
