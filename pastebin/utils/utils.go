package utils

import (
	// Go Builtin Packages
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	// Google Appengine Packages
	"appengine"
	"appengine/user"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"

	// The Gorilla Web Toolkit
	"github.com/gorilla/securecookie"
)

// http://andyrees.github.io/2015/your-code-a-mess-maybe-its-time-to-bring-in-the-decorators/
func ExtraSugar(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Ada", "*skips about* Hi! <3 ^_^")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; font-src netdna.bootstrapcdn.com fonts.gstatic.com; script-src 'self' netdna.bootstrapcdn.com ajax.googleapis.com linkhelp.clients.google.com www.google-analytics.com; style-src 'self' netdna.bootstrapcdn.com 'unsafe-inline'; img-src 'self' *; object-src 'none'; media-src 'none'")
		w.Header().Set("Strict-Transport-Security", "max-age=15552000")

		if strings.Contains(strings.ToLower(r.UserAgent()), "msie") {
			w.Header().Set("X-UA-Compatible", "IE=edge,chrome=1")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}

		f(w, r)
	}
}

func PanicOnErr(c appengine.Context, err error) {
	if err != nil {
		c.Errorf(err.Error())
		log.Panic(err)
	}
}

func ProcessForm(c appengine.Context, r *http.Request) {
	var err error
	if strings.Contains(strings.ToLower(r.Header.Get("content-type")), "multipart") {
		err = r.ParseMultipartForm(32 << 20) // 32 MB - http.defaultMaxMemory
	} else {
		err = r.ParseForm()
	}
	if err != nil {
		log.Panic(c, err)
	}
}

func SC() *securecookie.SecureCookie {
	CSRFAuthKey := []byte(os.Getenv("CSRFAuthKey"))
	EncryptionK := []byte(os.Getenv("EncryptionK"))
	sc := securecookie.New(CSRFAuthKey, EncryptionK)
	return sc
}

func TokenCookie(c appengine.Context, r *http.Request) (interface{}, bool) {
	// This presence of this cookie indicates we _may_ have an authorization
	// token for Google Drive available for the currently logged in user
	if cookie, err := r.Cookie("gdrive-token"); err == nil {
		value := make(map[string]string)
		if err = SC().Decode("gdrive-token", cookie.Value, &value); err != nil {
			return nil, false
		}
		lookietoken := make(map[string]interface{})
		if err = json.Unmarshal([]byte(value["gdrive-token"]), &lookietoken); err == nil {
			usr := user.Current(c)
			if usr != nil {
				if lookietoken["userid"] == usr.ID {
					return lookietoken["token"], true
				}
			}
		} else {
			return nil, false // invalid JSON or something lol
		}
	}

	return nil, false
}

func OAuthConfigDance(c appengine.Context) *oauth2.Config {
	GCPOAuthCID := []byte(os.Getenv("GCPOAuthCID"))
	config, err := google.ConfigFromJSON([]byte(GCPOAuthCID), drive.DriveAppdataScope)
	PanicOnErr(c, err)
	return config
}
