package auth

import (
	// Go Builtin Packages
	"log"
	"net/http"

	// Google Appengine Packages
	"appengine"
	"appengine/user"

	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"

	// Local Packages
	"pastebin/utils"
)

var Router *mux.Router

func init() {
	Router = mux.NewRouter()
	Router.HandleFunc("/pastebin/auth/login", utils.ExtraSugar(login)).Methods("GET").Name("login")
	Router.HandleFunc("/pastebin/auth/logout", utils.ExtraSugar(logout)).Methods("GET").Name("logout")

	Router.HandleFunc("/pastebin/auth/gdrive", utils.ExtraSugar(auth_gdrive_begin)).Methods("GET").Name("auth_gdrive_begin")
	Router.HandleFunc("/pastebin/auth/gdrive/complete", utils.ExtraSugar(auth_gdrive_complete)).Methods("GET").Name("auth_gdrive_complete")
}

func login(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	var dest string
	if dest = r.FormValue("next"); dest == "" {
		if dest = r.Referer(); dest == "" {
			dest = "/pastebin/"
		}
	}

	if usr := user.Current(c); usr != nil { // already logged in
		if dest != "/pastebin/login/" {
			http.Redirect(w, r, dest, http.StatusFound)
		} else {
			http.Redirect(w, r, "/pastebin/", http.StatusFound)
		}
	} else {
		if url, err := user.LoginURL(c, dest); err == nil {
			http.Redirect(w, r, url, http.StatusFound)
		} else {
			log.Panic(c, err)
		}
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	dest := r.Referer()

	if usr := user.Current(c); usr == nil { // already logged out
		if dest != "/pastebin/logout/" {
			http.Redirect(w, r, dest, http.StatusFound)
		} else {
			http.Redirect(w, r, "/pastebin/", http.StatusFound)
		}
	} else {
		if url, err := user.LogoutURL(c, dest); err == nil {
			http.Redirect(w, r, url, http.StatusFound)
		} else {
			log.Panic(c, err)
		}
	}
}
