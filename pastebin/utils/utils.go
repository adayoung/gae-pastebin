package utils

import (
	// Go Builtin Packages
	"context"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"

	// The Gorilla Web Toolkit
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

func Http404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	var tmpl = template.Must(template.ParseFiles("templates/404.tmpl"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Meep! We were trying to make the '404' page but something went wrong.", http.StatusInternalServerError)
	}
}

// http://andyrees.github.io/2015/your-code-a-mess-maybe-its-time-to-bring-in-the-decorators/
func ExtraSugar(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "staticDomain", os.Getenv("StaticDomain"))
		r = r.WithContext(ctx)

		w.Header().Set("Ada", "*skips about* Hi! <3 ^_^")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self' "+os.Getenv("StaticDomain")+"; font-src netdna.bootstrapcdn.com fonts.gstatic.com; script-src 'self' netdna.bootstrapcdn.com ajax.googleapis.com linkhelp.clients.google.com www.google-analytics.com cdnjs.cloudflare.com https://www.google.com/recaptcha/ https://www.gstatic.com/recaptcha/ "+os.Getenv("StaticDomain")+"; style-src 'self' netdna.bootstrapcdn.com cdnjs.cloudflare.com "+os.Getenv("StaticDomain")+" 'unsafe-inline'; img-src 'self' *; object-src 'none'; media-src 'none'; connect-src 'self' *.googleusercontent.com; frame-src 'self' "+os.Getenv("StaticDomain")+" blob: https://www.google.com/recaptcha/; frame-ancestors 'none'")
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
			Secure:   os.Getenv("CSRFSecureC") == "true",
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

func ProcessForm(r *http.Request) error {
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

func OAuthConfigDance() (*oauth2.Config, error) {
	GCPOAuthCID := []byte(os.Getenv("GCPOAuthCID"))
	if config, err := google.ConfigFromJSON([]byte(GCPOAuthCID), drive.DriveFileScope); err == nil {
		return config, nil
	} else {
		return config, err
	}
}

type reCaptchaResponse struct {
	Success bool    `json:"success"`
	Score   float64 `json:"score,number"`
}

func ValidateCaptcha(recaptchaResponse string, remoteip string) (float64, error) {
	gRecaptchaRequest := url.Values{}
	gRecaptchaRequest.Add("secret", os.Getenv("ReCAPTCHASecrt"))
	gRecaptchaRequest.Add("response", recaptchaResponse)
	gRecaptchaRequest.Add("remoteip", remoteip)

	if !(len(recaptchaResponse) > 0) {
		log.Println("WARNING: missing reCAPTCHA token")
		return 0.0, nil
	}

	rvCall := &http.Client{}
	if response, err := rvCall.PostForm("https://www.google.com/recaptcha/api/siteverify", gRecaptchaRequest); err == nil {
		defer response.Body.Close()
		if rContent, err := ioutil.ReadAll(response.Body); err == nil {
			rvResponse := &reCaptchaResponse{}
			if err := json.Unmarshal(rContent, &rvResponse); err == nil {
				if rvResponse.Success {
					return rvResponse.Score, nil
				} else {
					log.Println("WARNING: invalid reCAPTCHA token")
					return 0.0, nil
				}
			} else {
				return 0.0, err
			}
		} else {
			return 0.0, err
		}
	} else {
		return 0.0, err
	}
}
