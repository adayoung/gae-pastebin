package api_v1

import (
	// Go Builtin Packages
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/user"

	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"

	// Local Packages
	"pastebin/models"
	"pastebin/utils"
)

var API_Router *mux.Router

func init() {
	API_Router = mux.NewRouter()
	API_Router.HandleFunc("/pastebin/api/v1/", utils.ExtraSugar(welcome)).Methods("GET").Name("apiwelcome")
	API_Router.HandleFunc("/pastebin/api/v1/echo", utils.ExtraSugar(echo)).Methods("GET").Name("echo")
	API_Router.HandleFunc("/pastebin/api/v1/create", utils.ExtraSugar(create)).Methods("POST").Name("create")
}

func welcome(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	usr := user.Current(c)
	token, _ := gencookie(usr)
	var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/api_v1.tmpl"))
	if err := tmpl.Execute(w, map[string]interface{}{
		"token": token,
		"user":  usr,
		"rkey":  os.Getenv("ReCAPTCHAKey"),
	}); err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to assemble the API Key page but something went wrong.", http.StatusInternalServerError)
		return
	}
}

func gencookie(usr *user.User) (string, error) {
	token := make(map[string]string)
	if usr != nil {
		token["user"] = usr.ID
	} else {
		token["user"] = ""
	}
	token["timestamp"] = time.Now().Format(time.StampNano)
	return utils.SC().Encode("auth-token", token)
}

func echo(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	usr := user.Current(c)
	t, _ := gencookie(usr)
	w.Write([]byte(t))
}

func create(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	if err := utils.ProcessForm(c, r); err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to process your input but something went wrong.", http.StatusInternalServerError)
		return
	}

	auth_token := make(map[string]string)
	received_token := strings.TrimSpace(r.Form.Get("auth"))
	if err := utils.SC().Decode("auth-token", received_token, &auth_token); err != nil {
		c.Warningf("API call rejected, received_token -> " + received_token)
		at_url := r.URL.Scheme + "://" + r.URL.Host + "/pastebin/api/v1/echo"
		http.Error(w, "Auth token invalid/not supplied, you can get one here: "+at_url, http.StatusUnauthorized)
		return
	}

	paste_id, err := models.NewPaste(c, r, 0.0)
	if err != nil {
		if _, ok := err.(*models.ValidationError); ok {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else {
			http.Error(w, "BARF!@ Something's broken!@%", http.StatusInternalServerError)
			return
		}
	}

	// Is that a good idea? O_o I dunno :<
	http.Redirect(w, r, r.URL.Scheme+"://"+r.URL.Host+"/pastebin/"+paste_id, http.StatusSeeOther)
	w.Write([]byte(r.URL.Scheme + "://" + r.URL.Host + "/pastebin/" + paste_id))
}
