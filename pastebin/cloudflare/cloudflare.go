package cloudflare

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

var Token, Domain, PageURL, Schema, PurgeAPI string

func InitCF(token, zoneid, domain, pageurl, schema, purgeapi string) {
	PurgeAPI = fmt.Sprintf(purgeapi, zoneid)
	Token = token
	Domain = domain
	PageURL = pageurl
	Schema = schema
}

func Purge(pasteID string) {
	purgeURL := Schema + "://" + Domain + fmt.Sprintf(PageURL, pasteID)

	requestData := map[string]interface{}{
		"files": []map[string]string{
			{"url": purgeURL},
		},
	}

	var requestBuffer bytes.Buffer
	encodedRequest := bufio.NewWriter(&requestBuffer)
	json.NewEncoder(encodedRequest).Encode(requestData)
	encodedRequest.Flush()

	client := &http.Client{}
	request, err := http.NewRequest("POST", PurgeAPI, &requestBuffer)
	if err != nil {
		log.Printf("ERROR: cloudflare.Purge, NewRequest: %v", err)
		return
	}

	request.Header.Set("Authorization", "Bearer "+Token)
	if _, err := client.Do(request); err != nil {
		log.Printf("ERROR: cloudflare.Purge, Do: %v", err)
	}
}
