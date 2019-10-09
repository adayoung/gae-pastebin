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

type config struct {
	WebApp struct {
		CSRFAuthKey  string `yaml:"CSRFAuthKey"`
		CSRFSecureC  string `yaml:"CSRFSecureC"`
		EncryptionK  string `yaml:"EncryptionK"`
		ListenPort   string `yaml:"ListenPort"`
		StaticDomain string `yaml:"StaticDomain"`
	}

	GoogleDrive struct {
		GCPOAuthCID string `yaml:"GCPOAuthCID"`
	}

	ReCAPTCHA struct {
		Key    string
		Secret string
	}

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
	var _config config
	if data, err := ioutil.ReadFile("config.yaml"); err == nil {
		if err = yaml.Unmarshal([]byte(data), &_config); err == nil {
			// FIXME: os.Setenv may emit an error which is currently unhandled
			os.Setenv("CSRFAuthKey", _config.WebApp.CSRFAuthKey)
			os.Setenv("CSRFSecureC", _config.WebApp.CSRFSecureC)
			os.Setenv("EncryptionK", _config.WebApp.EncryptionK)
			os.Setenv("StaticDomain", _config.WebApp.StaticDomain)
			os.Setenv("GCPOAuthCID", _config.GoogleDrive.GCPOAuthCID)
			os.Setenv("ReCAPTCHAKey", _config.ReCAPTCHA.Key)
			os.Setenv("ReCAPTCHASecrt", _config.ReCAPTCHA.Secret)
		} else {
			log.Println("ERROR: Error with parsing config.yaml.")
			log.Fatalf("ERROR: %v", err)
		}
	} else {
		log.Println("ERROR: The file 'config.yaml' could not be read.")
		log.Fatalf("ERROR: %v", err)
	}

	if err := storage.InitDB(_config.Database.Connection); err != nil {
		log.Println("ERROR: The database could not be initialized, DB will not unavailable.")
		log.Fatalf("ERROR: %v", err)
	}

	utils.InitRedisPool(_config.Database.Redis)
	cloudflare.InitCF(
		_config.CloudFlare.Token, _config.CloudFlare.ZoneID,
		_config.CloudFlare.Domain, _config.CloudFlare.PageURL,
		_config.CloudFlare.Schema, _config.CloudFlare.PurgeAPI,
	)

	// Router begins here
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", index).Methods("GET").Name("index")
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	r.NotFoundHandler = http.HandlerFunc(utils.Http404)

	pastebin.InitRoutes(r)

	http.Handle("/", r)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+_config.WebApp.ListenPort, nil))
}

func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/pastebin/", http.StatusFound)
	// var tmpl = template.Must(template.ParseFiles("templates/base.tmpl"))
	// if err := tmpl.Execute(w, nil); err != nil {
	// 	http.Error(w, "Meep! We were trying to make the 'base' page but something went wrong.", http.StatusInternalServerError)
	// }
}
