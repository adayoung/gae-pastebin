package cloudflare

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"

	"github.com/adayoung/gae-pastebin/pastebin/utils"
)

var Token, Domain, PageURL, Schema, PurgeAPI string
var cfMutex sync.Mutex

func InitCF(token, zoneid, domain, pageurl, schema, purgeapi string) {
	PurgeAPI = fmt.Sprintf(purgeapi, zoneid)
	Token = token
	Domain = domain
	PageURL = pageurl
	Schema = schema

	ticker := time.NewTicker(time.Hour)
	go func() {
		for range ticker.C {
			conn := utils.RedisPool.Get()
			doPurge(conn, true)
		}
	}()
}

func Purge(pasteID string) {
	conn := utils.RedisPool.Get()

	if _, err := conn.Do("RPUSH", "CFDelQueue", pasteID); err != nil {
		log.Printf("ERROR: Delete queue operation failed for %s, %v", pasteID, err)
	}

	doPurge(conn, false)
}

func doPurge(conn redis.Conn, doItNow bool) {
	const maxQueueLength = 30

	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("ERROR: CF-Redis connection closure failed, %v", err)
		}
	}()

	if qLength, err := redis.Int(conn.Do("LLEN", "CFDelQueue")); err != nil {
		log.Printf("ERROR: Could not retrieve delete queue length, %v", err)
	} else if qLength > maxQueueLength || doItNow {
		cfMutex.Lock()
		defer cfMutex.Unlock()

		var delPasteIDs []string
		var delURLs []map[string]string
		for c := 0; c < maxQueueLength; c++ {
			if c >= qLength { // because doItNow
				break
			}

			if item, err := redis.String(conn.Do("LPOP", "CFDelQueue")); err != nil {
				log.Printf("ERROR: Eep CFDelQueue returned a non string? %v", err)
			} else {
				delPasteIDs = append(delPasteIDs, item)
				delURLs = append(delURLs, map[string]string{
					"url": Schema + "://" + Domain + fmt.Sprintf(PageURL, item),
				})
			}
		}

		if qLength > 0 {
			log.Printf("INFO: About to purge the following pastes, %v", delPasteIDs)
			requestData := map[string]interface{}{
				"files": delURLs,
			}

			var requestBuffer bytes.Buffer
			encodedRequest := bufio.NewWriter(&requestBuffer)
			if err := json.NewEncoder(encodedRequest).Encode(requestData); err != nil {
				log.Printf("ERROR: Meep we couldn't encode a request for Cloudflare, %v\n", err)
			} else {
				encodedRequest.Flush()

				client := &http.Client{}
				request, err := http.NewRequest("POST", PurgeAPI, &requestBuffer)
				if err != nil {
					log.Printf("ERROR: cloudflare.Purge, NewRequest: %v", err)
				} else {
					request.Header.Set("Authorization", "Bearer "+Token)
					if response, err := client.Do(request); err != nil {
						log.Printf("ERROR: cloudflare.doPurge, Do: %v", err)
					} else if response.StatusCode != 200 {
						defer response.Body.Close()
						if data, err := ioutil.ReadAll(response.Body); err != nil {
							log.Printf("ERROR: cloudflare.doPurge returned non-OK, data could not be read, %v\n", err)
						} else {
							log.Printf("ERROR: cloudflare.doPurge returned non-OK, %s\n", string(data))
							// TODO: Requeue failed pasteIDs again for cache purge
						}
					}
				}
			}
		}
	}
}
