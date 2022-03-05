package utils

import (
	// Go Builtin Packages
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

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
		staticDomain := os.Getenv("StaticDomain")
		ctx := context.WithValue(r.Context(), "staticDomain", staticDomain)

		if session, err := sessionStore().Get(r, "_app_session"); err != nil {
			log.Printf("WARNING: sessionStore.Get call failed for _app_session, %v", err)
			ClearAppSession(w)
		} else {
			userID := session.Values["userID"]
			if userID != nil {
				ctx = context.WithValue(ctx, "userID", userID)
				if err := InitAppSession(w, r, userID.(string), true); err != nil {
					log.Printf("WARNING: InitAppSession call failed in context, %v", err)
				}
			}
		}

		r = r.WithContext(ctx)

		w.Header().Set("Ada", "*skips about* Hi! <3 ^_^")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", getCSP(staticDomain))
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")

		if strings.Contains(strings.ToLower(r.UserAgent()), "msie") {
			w.Header().Set("X-UA-Compatible", "IE=edge,chrome=1")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}

		f(w, r)
	}
}

func sessionStore() *sessions.CookieStore {
	CSRFAuthKey := []byte(os.Getenv("CSRFAuthKey"))
	EncryptionK := []byte(os.Getenv("EncryptionK"))
	ss := sessions.NewCookieStore(CSRFAuthKey, EncryptionK)
	return ss
}

func UpdateSession(w http.ResponseWriter, r *http.Request, paste_id string, remove bool) error {
	if session, err := sessionStore().Get(r, "_pb_session"); err != nil {
		return err
	} else {
		session.Options = &sessions.Options{
			Path:     "/pastebin/",
			MaxAge:   0,
			HttpOnly: true,
			Secure:   os.Getenv("CSRFSecureC") == "true",
			SameSite: http.SameSiteStrictMode,
		}

		if remove {
			_paste_id := paste_id[1:]
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

		return session.Save(r, w)
	}
}

func CheckSession(r *http.Request, paste_id string) (bool, error) {
	if session, err := sessionStore().Get(r, "_pb_session"); err != nil {
		return false, err
	} else {
		if session.Values[paste_id] != nil {
			return true, nil
		}
	}
	return false, nil
}

func InitAppSession(w http.ResponseWriter, r *http.Request, userID string, refresh bool) error {
	if session, err := sessionStore().Get(r, "_app_session"); err != nil {
		return err
	} else {
		session.Options = &sessions.Options{
			Path:     "/pastebin/",
			MaxAge:   86400 * 3,
			HttpOnly: true,
			Secure:   os.Getenv("CSRFSecureC") == "true",
		}

		if !refresh {
			userID = fmt.Sprintf("%x", sha256.Sum256([]byte(userID)))
		}

		session.Values["userID"] = userID
		return session.Save(r, w)
	}
}

func ClearAppSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Path:     "/pastebin/",
		Name:     "_app_session",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("CSRFSecureC") == "true",
		SameSite: http.SameSiteStrictMode,
	})
}

func ClearOauthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{ // That was a HALF A KILO cookie!! :O
		Path:     "/pastebin/",
		Name:     "_oauth2_gdrive",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("CSRFSecureC") == "true",
		SameSite: http.SameSiteStrictMode,
	})
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

func OAuthConfigDance(provider string, redirectURL string, scopes ...string) (*oauth2.Config, error) {
	if provider == "google" {
		GCPOAuthCID := []byte(os.Getenv("GCPOAuthCID"))
		if config, err := google.ConfigFromJSON([]byte(GCPOAuthCID), scopes...); err == nil {
			config.RedirectURL = redirectURL
			return config, nil
		} else {
			return config, err
		}
	} else if provider == "github" {
		GitHubClientID := os.Getenv("GitHubClientID")
		GitHubClientSecret := os.Getenv("GitHubClientSecret")
		return &oauth2.Config{
			ClientID:     GitHubClientID,
			ClientSecret: GitHubClientSecret,
			Endpoint:     github.Endpoint,
			RedirectURL:  redirectURL,
		}, nil
	} else if provider == "discord" {
		DiscordClientID := os.Getenv("DiscordClientID")
		DiscordClientSecret := os.Getenv("DiscordClientSecret")
		return &oauth2.Config{
			ClientID:     DiscordClientID,
			ClientSecret: DiscordClientSecret,
			// https://discordapp.com/developers/docs/topics/oauth2#shared-resources-oauth2-urls
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discordapp.com/api/oauth2/authorize",
				TokenURL: "https://discordapp.com/api/oauth2/token",
			},
			RedirectURL: redirectURL,
			Scopes:      scopes,
		}, nil
	}
	return nil, fmt.Errorf("No supported provider specified for oauth config")
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
