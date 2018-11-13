package utils

import (
	// Go Builtin Packages
	"net/http"
	"os"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"

	// The Gorilla Web Toolkit
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// http://andyrees.github.io/2015/your-code-a-mess-maybe-its-time-to-bring-in-the-decorators/
func ExtraSugar(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Ada", "*skips about* Hi! <3 ^_^")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; font-src netdna.bootstrapcdn.com fonts.gstatic.com; script-src 'self' netdna.bootstrapcdn.com ajax.googleapis.com linkhelp.clients.google.com www.google-analytics.com cdnjs.cloudflare.com; style-src 'self' netdna.bootstrapcdn.com cdnjs.cloudflare.com 'unsafe-inline'; img-src 'self' *; object-src 'none'; media-src 'none'; connect-src 'self' *.googleusercontent.com; frame-src 'self' blob:; frame-ancestors 'none'")
		w.Header().Set("Strict-Transport-Security", "max-age=15552000")

		if strings.Contains(strings.ToLower(r.UserAgent()), "msie") {
			w.Header().Set("X-UA-Compatible", "IE=edge,chrome=1")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}

		f(w, r)
	}
}

var sessionStore = sessions.NewCookieStore([]byte(os.Getenv("CSRFAuthKey")))

func UpdateSession(w http.ResponseWriter, r *http.Request, paste_id string, remove bool) error {
	if session, err := sessionStore.Get(r, "_pb_session"); err != nil {
		return err
	} else {
		session.Options = &sessions.Options{
			Path:     "/pastebin/",
			MaxAge:   0,
			HttpOnly: true,
			Secure:   !appengine.IsDevAppServer(),
		}

		if remove == true {
			_paste_id := paste_id[1:len(paste_id)]
			delete(session.Values, _paste_id)
		} else {
			session.Values[paste_id] = time.Now().Format(time.RFC3339)
			if len(session.Values) > 10 { // remember up to 10 pastes only
				var popindex string
				_time := time.Now().Format(time.RFC3339)
				for key, value := range session.Values {
					if value.(string) < _time {
						_time = value.(string)
						popindex = key.(string)
					}
				}
				delete(session.Values, popindex)
			}
		}

		err = session.Save(r, w)
		if err != nil {
			return err
		}
	}
	return nil
}

func CheckSession(r *http.Request, paste_id string) (bool, error) {
	if session, err := sessionStore.Get(r, "_pb_session"); err != nil {
		return false, err
	} else {
		if session.Values[paste_id] != nil {
			return true, nil
		}
	}
	return false, nil
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
