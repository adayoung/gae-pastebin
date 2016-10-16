package auth

import (
	// Go Builtin Packages
	"html/template"
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

	Router.HandleFunc("/pastebin/auth/gdrive/start", utils.ExtraSugar(auth_gdrive_start)).Methods("GET").Name("auth_gdrive_start")
	Router.HandleFunc("/pastebin/auth/gdrive/finish", utils.ExtraSugar(auth_gdrive_finish)).Methods("GET").Name("auth_gdrive_finish")
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
			var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/auth.tmpl"))
			if err := tmpl.Execute(w, map[string]interface{}{
				csrf.TemplateTag: csrf.TemplateField(r),
				"dest":           dest,
				"user":           user.Current(c),
			}); err != nil {
				c.Errorf(err.Error())
				http.Error(w, "Meep! We were trying to make the 'login' page but something went wrong.", http.StatusInternalServerError)
				return
			}
			return
		}

		if strings.HasSuffix(dest, "/pastebin/auth/login") {
			http.Redirect(w, r, "/pastebin/", http.StatusFound)
			return
		} else {
			http.Redirect(w, r, dest, http.StatusFound)
			return
		}
	} else {
		if url, err := user.LoginURL(c, dest); err == nil {
			http.Redirect(w, r, url, http.StatusFound)
			return
		} else {
			c.Errorf(err.Error())
			http.Error(w, "Meep! We were trying to make the 'login' url but something went wrong.", http.StatusInternalServerError)
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
			return
		} else {
			http.Redirect(w, r, "/pastebin/", http.StatusFound)
			return
		}
	} else {
		if url, err := user.LogoutURL(c, dest); err == nil {
			http.Redirect(w, r, url, http.StatusFound)
		} else {
			c.Errorf(err.Error())
			http.Error(w, "Meep! We were trying to make the 'logout' url but something went wrong.", http.StatusInternalServerError)
		}
	}
}
