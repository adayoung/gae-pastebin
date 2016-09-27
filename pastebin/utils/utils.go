package utils

import (
	// Go Builtin Packages
	"log"
	"net/http"
	"strings"

	// The Gorilla Web Toolkit
	"github.com/gorilla/context"
	"github.com/gorilla/securecookie"

	// nu7hatch/gouuid / >_<
	"github.com/nu7hatch/gouuid"
)

var hashKey = []byte("very-secret")
var blockKey = []byte("a-lot-secreta-lot-secreta-lot-se")
var sc = securecookie.New(hashKey, blockKey)

func Autograph(w http.ResponseWriter) (http.ResponseWriter, string) {
	t_value, _ := uuid.NewV4()
	values := map[string]*uuid.UUID{
		"token" : t_value, // TODO: replace this with uuid_v4
	}

	enc_val, err := sc.Encode("_xsrf_token", values)
	if err == nil {
		cookie := &http.Cookie{
			Name:     "_xsrf_token",
			Value:    enc_val,
			Path:     "/",
			Secure:   false,
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
	} else {
		log.Fatal(err)
	}

	return w, enc_val
}

// http://andyrees.github.io/2015/your-code-a-mess-maybe-its-time-to-bring-in-the-decorators/
func ExtraSugar(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Ada", "*skips about* Hi! <3 ^_^")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		// w.Header().Set("Strict-Transport-Security", "max-age=15552000")

		if strings.Contains(strings.ToLower(r.UserAgent()), "msie") {
			w.Header().Set("X-UA-Compatible", "IE=edge,chrome=1")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}

		// Get/Set signed cookie for CSRF prevention
		var _xsrf_token string
		if cookie, err := r.Cookie("_xsrf_token"); err == nil {
			values := make(map[string]*uuid.UUID)
			if err := sc.Decode("_xsrf_token", cookie.Value, &values); err == nil {
				_xsrf_token = cookie.Value
			} else {
				// Invalid cookie / Set cookie here
				log.Println("Huh? COokie invalid? O_o")
				log.Println(err)
				w, _xsrf_token = Autograph(w)
			}
		} else {
			// No cookie / Set cookie here
			w, _xsrf_token = Autograph(w)
		}

		context.Set(r, "_xsrf_token", _xsrf_token)
		f(w, r)
	}
}
