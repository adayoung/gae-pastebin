package api_v1

import (
	// Go Builtin Packages
	"log"
	"net/http"
	"os"
	"time"

	// Google Appengine Packages
	"appengine"

	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"

	// Local Packages
	"pastebin/models"
	"pastebin/utils"
)

var API_Router *mux.Router

func init() {
	API_Router = mux.NewRouter()
	API_Router.HandleFunc("/pastebin/api/v1/echo", utils.ExtraSugar(echo)).Methods("GET").Name("echo")
	API_Router.HandleFunc("/pastebin/api/v1/create", utils.ExtraSugar(create)).Methods("POST").Name("create")
}

func echo(w http.ResponseWriter, r *http.Request) {
	CSRFAuthKey := []byte(os.Getenv("CSRFAuthKey"))
	EncryptionK := []byte(os.Getenv("EncryptionK"))
	sc := securecookie.New(CSRFAuthKey, EncryptionK)
	t, _ := sc.Encode("auth-token", time.Now().Format(time.StampNano))
	w.Write([]byte(t))
}

func create(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	CSRFAuthKey := []byte(os.Getenv("CSRFAuthKey"))
	EncryptionK := []byte(os.Getenv("EncryptionK"))
	sc := securecookie.New(CSRFAuthKey, EncryptionK)

	log.Printf("Hello, world! :D")
	if err := r.ParseForm(); err != nil {
		log.Panic(c, err)
	}

	if !(len(r.PostForm.Get("content")) > 0) {
		http.Error(w, "Oops, we need 'content' for this.", http.StatusBadRequest)
		return
	}

	var auth_token string
	if err := sc.Decode("auth-token", r.PostForm.Get("auth"), &auth_token); err != nil {
		at_url := r.URL.Scheme + "://" + r.URL.Host + "/pastebin/api/v1/echo"
		http.Error(w, "Auth token invalid/not supplied, you can get one here: " + at_url, http.StatusUnauthorized)
		return
	}

	paste_id, err := models.NewPaste(c, r)
	if err != nil {
		if _, ok := err.(models.ValidationError); !ok {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			http.Error(w, "BARF!@ Something's broken!@%", http.StatusInternalServerError)
			return
		}
	}

	w.Write([]byte(r.URL.Scheme + "://" + r.URL.Host + "/pastebin/" + paste_id))
}
