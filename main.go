package main

import (
	// Go Builtin Packages
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v2"

	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"

	// Local Packages
	"github.com/adayoung/gae-pastebin/pastebin"
	"github.com/adayoung/gae-pastebin/pastebin/cloudflare"
	"github.com/adayoung/gae-pastebin/pastebin/utils"
	"github.com/adayoung/gae-pastebin/pastebin/utils/storage"
)

type envKeys struct {
	CSRFAuthKey    string `yaml:"CSRFAuthKey"`
	CSRFSecureC    string `yaml:"CSRFSecureC"`
	EncryptionK    string `yaml:"EncryptionK"`
	GCPOAuthCID    string `yaml:"GCPOAuthCID"`
	ReCAPTCHAKey   string `yaml:"ReCAPTCHAKey"`
	ReCAPTCHASecrt string `yaml:"ReCAPTCHASecrt"`
	StaticDomain   string `yaml:"StaticDomain"`
	ListenPort     string `yaml:"ListenPort"`

	Database struct {
		Connection string
		Redis      string
	}

	CloudFlare struct {
		Token    string
		ZoneID   string
		Schema   string
		Domain   string
		PageURL  string
		PurgeAPI string
	}
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
			os.Setenv("StaticDomain", _envKeys.StaticDomain)
		} else {
			log.Println("ERROR: Error with parsing keys.yaml.")
			log.Fatalf("ERROR: %v", err)
		}
	} else {
		log.Println("ERROR: The file 'keys.yaml' could not be read.")
		log.Fatalf("ERROR: %v", err)
	}

	if err := storage.InitDB(_envKeys.Database.Connection); err != nil {
		log.Println("ERROR: The database could not be initialized, DB will not unavailable.")
		log.Fatalf("ERROR: %v", err)
	}

	utils.InitRedisPool(_envKeys.Database.Redis)
	cloudflare.InitCF(
		_envKeys.CloudFlare.Token, _envKeys.CloudFlare.ZoneID,
		_envKeys.CloudFlare.Domain, _envKeys.CloudFlare.PageURL,
		_envKeys.CloudFlare.Schema, _envKeys.CloudFlare.PurgeAPI,
	)

	// Router begins here
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", index).Methods("GET").Name("index")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.NotFoundHandler = http.HandlerFunc(utils.Http404)

	pastebin.InitRoutes(r)

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+_envKeys.ListenPort, nil))
}

func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/pastebin/", http.StatusFound)
	// var tmpl = template.Must(template.ParseFiles("templates/base.tmpl"))
	// if err := tmpl.Execute(w, nil); err != nil {
	// 	http.Error(w, "Meep! We were trying to make the 'base' page but something went wrong.", http.StatusInternalServerError)
	// }
}
