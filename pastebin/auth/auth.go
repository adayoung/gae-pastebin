package auth

import (
	// Go Builtin Packages
	"html/template"
	"log"
	"net/http"
	"strings"

	// Google Appengine Packages
	"appengine"
	"appengine/user"

	// The Gorilla Web Toolkit
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	// Local Packages
	"pastebin/utils"
)

var Router *mux.Router

func init() {
	Router = mux.NewRouter()
	Router.HandleFunc("/pastebin/auth/login", utils.ExtraSugar(login)).Methods("GET", "POST").Name("login")
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
		if r.Method == "GET" {
			var tmpl = template.Must(template.ParseFiles("templates/base.tmpl","pastebin/templates/auth.tmpl"))
			if err := tmpl.Execute(w, map[string]interface{}{
				csrf.TemplateTag: csrf.TemplateField(r),
				"dest": dest,
				"user": user.Current(c),
			}); err != nil {
				log.Panic(c, err)
			}
			return
		}

		if strings.HasSuffix(dest, "/pastebin/auth/login") {
			http.Redirect(w, r, "/pastebin/", http.StatusFound)
		} else {
			http.Redirect(w, r, dest, http.StatusFound)
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
	var dest string
	if dest = r.FormValue("next"); dest == "" {
		if dest = r.Referer(); dest == "" {
			dest = "/pastebin/"
		}
	}

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
