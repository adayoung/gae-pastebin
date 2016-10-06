package api_v1

import (
	// Go Builtin Packages
	"log"
	"net/http"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"

	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"

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
	t, _ := utils.SC().Encode("auth-token", time.Now().Format(time.StampNano))
	w.Write([]byte(t))
}

func create(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	utils.ProcessForm(c, r)
	var auth_token string
	received_token := strings.TrimSpace(r.Form.Get("auth"))
	if err := utils.SC().Decode("auth-token", received_token, &auth_token); err != nil {
		c.Warningf("API call rejected, received_token -> " + received_token)
		log.Print(c, err)
		at_url := r.URL.Scheme + "://" + r.URL.Host + "/pastebin/api/v1/echo"
		http.Error(w, "Auth token invalid/not supplied, you can get one here: "+at_url, http.StatusUnauthorized)
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

	// Is that a good idea? O_o I dunno :<
	http.Redirect(w, r, r.URL.Scheme + "://" + r.URL.Host + "/pastebin/" + paste_id, http.StatusSeeOther)
	w.Write([]byte(r.URL.Scheme + "://" + r.URL.Host + "/pastebin/" + paste_id))
}
