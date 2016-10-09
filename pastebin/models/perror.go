package models

import (
	// Go Builtin Packages
	"fmt"
	"regexp"
	"strconv"
	"strings"

	// Google Appengine Packages
	"appengine"
	"appengine/datastore"
)

type GDriveAPIError struct {
	Code     int    // The response code we received
	Response string // The response text we received
}

func (e *GDriveAPIError) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Response)
}

func parseAPIError(c appengine.Context, rerr error, p *Paste) error {
	// THIS! Because the upstream API won't give structured errors!@
	serr := rerr.Error()
	perr := &GDriveAPIError{}
	token_revoked := false
	var message string
	var code int

	if strings.HasPrefix(serr, "googleapi: got HTTP response code ") {
		// https://github.com/google/google-api-go-client/blob/3cf64a039723963488f603d140d0aec154fdcd20/googleapi/googleapi.go#L87
		code_regex := regexp.MustCompile("^(?s)googleapi: got HTTP response code ([0-9]{3}) with body: (.+)")
		code_found := code_regex.FindStringSubmatch(serr)[1:]
		lcode, cerr := strconv.Atoi(code_found[0])
		if cerr != nil { // BARF!@
			return rerr
		}

		code = lcode
		message = code_found[1]
	}

	if strings.HasPrefix(serr, "googleapi: Error ") {
		// https://github.com/google/google-api-go-client/blob/3cf64a039723963488f603d140d0aec154fdcd20/googleapi/googleapi.go#L90
		code_regex := regexp.MustCompile("^Error ([0-9]{3}): (.+)")
		code_found := code_regex.FindStringSubmatch(serr)[1:]
		lcode, cerr := strconv.Atoi(code_found[0])
		if cerr != nil { // BARF!@
			return rerr
		}

		code = lcode
		message = code_found[1]

		if code == 401 {
			token_revoked = true
		}
	}

	if strings.Contains(serr, ": oauth2: cannot fetch token: ") {
		// https://github.com/golang/oauth2/blob/1e695b1c8febf17aad3bfa7bf0a819ef94b98ad5/internal/token.go#L177
		code_regex := regexp.MustCompile("(?s)cannot fetch token: ([0-9]{3}).+?Response: (.+)")
		code_found := code_regex.FindStringSubmatch(serr)[1:]
		lcode, cerr := strconv.Atoi(code_found[0])
		if cerr != nil { // BARF!@
			return rerr
		}

		code = lcode
		message = code_found[1]

		if strings.Contains(message, "Token has been revoked.") {
			token_revoked = true
		}
	}

	perr.Code = code
	perr.Response = message

	if perr.Code == 404 { // Oops, the paste's contents have disappeared from upstream
		p.Delete(c) // No err here, we just want to get rid of it xD
	}

	if token_revoked == true { // Oops, our access has been revoked
		key := datastore.NewKey(c, "OAuthToken", p.UserID, 0, nil)
		datastore.Delete(c, key) // No err here, we just want to get rid of it xD
	}

	if len(perr.Response) > 0 {
		return perr
	} else {
		c.Warningf("-flails- We couldn't parse the API error from upstream!")
		c.Errorf(rerr.Error())
		return rerr
	}
}
