package utils

import (
	// Go Builtin Packages
	"net/http"
	"os"
	"strings"

	// Google Appengine Packages
	"appengine"

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

func ProcessForm(c appengine.Context, r *http.Request) error {
	var err error
	if strings.Contains(strings.ToLower(r.Header.Get("content-type")), "multipart") {
		err = r.ParseMultipartForm(32 << 20) // 32 MB - http.defaultMaxMemory
	} else {
		err = r.ParseForm()
	}
	return err
}

func SC() *securecookie.SecureCookie {
	CSRFAuthKey := []byte(os.Getenv("CSRFAuthKey"))
	EncryptionK := []byte(os.Getenv("EncryptionK"))
	sc := securecookie.New(CSRFAuthKey, EncryptionK)
	return sc
}

func OAuthConfigDance(c appengine.Context) (*oauth2.Config, error) {
	GCPOAuthCID := []byte(os.Getenv("GCPOAuthCID"))
	if config, err := google.ConfigFromJSON([]byte(GCPOAuthCID), drive.DriveFileScope); err == nil {
		return config, nil
	} else {
		return config, err
	}
}
