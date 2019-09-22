package main

import (
	// Go Builtin Packages
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v2"

	// The Gorilla Web Toolkit
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	// Local Packages
	"github.com/adayoung/gae-pastebin/pastebin"
)

type envKeys struct {
	CSRFAuthKey    string `yaml:"CSRFAuthKey"`
	CSRFSecureC    string `yaml:"CSRFSecureC"`
	EncryptionK    string `yaml:"EncryptionK"`
	GCPOAuthCID    string `yaml:"GCPOAuthCID"`
	ReCAPTCHAKey   string `yaml:"ReCAPTCHAKey"`
	ReCAPTCHASecrt string `yaml:"ReCAPTCHASecrt"`
}

func main() {
	// Environment formerly set from keys.yaml by AppEngine
	var _envKeys envKeys
	if data, err := ioutil.ReadFile("keys.yaml"); err == nil {
		if err = yaml.Unmarshal([]byte(data), &_envKeys); err == nil {
			// FIXME: os.Setenv may emit an error which is currently unhandled
			os.Setenv("CSRFAuthKey", _envKeys.CSRFAuthKey)
			os.Setenv("CSRFSecureC", _envKeys.CSRFSecureC)
			os.Setenv("EncryptionK", _envKeys.EncryptionK)
			os.Setenv("GCPOAuthCID", _envKeys.GCPOAuthCID)
			os.Setenv("ReCAPTCHAKey", _envKeys.ReCAPTCHAKey)
			os.Setenv("ReCAPTCHASecrt", _envKeys.ReCAPTCHASecrt)
		} else {
			log.Println("ERROR: Error with parsing keys.yaml.")
			log.Fatalf("error: %v", err)
		}
	} else {
		log.Println("ERROR: The file 'keys.yaml' could not be read.")
		log.Fatalf("error: %v", err)
	}

	// Router begins here
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", index).Methods("GET").Name("index")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.NotFoundHandler = http.HandlerFunc(Http404)

	pastebin.InitRoutes(r)

	CSRFAuthKey := os.Getenv("CSRFAuthKey")
	CSRF := csrf.Protect([]byte(CSRFAuthKey), csrf.Secure(os.Getenv("CSRFSecureC") == "true"))

	log.Fatal(http.ListenAndServe("127.0.0.1:2019", CSRF(r)))
}

func Http404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	var tmpl = template.Must(template.ParseFiles("templates/404.tmpl"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Meep! We were trying to make the '404' page but something went wrong.", http.StatusInternalServerError)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/pastebin/", http.StatusFound)
	// var tmpl = template.Must(template.ParseFiles("templates/base.tmpl"))
	// if err := tmpl.Execute(w, nil); err != nil {
	// 	http.Error(w, "Meep! We were trying to make the 'base' page but something went wrong.", http.StatusInternalServerError)
	// }
}
