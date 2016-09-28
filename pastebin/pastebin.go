package pastebin

import (
	// Go Builtin Packages
	"html/template"
	"net/http"
	"log"

	// Google Appengine Packages
	// "appengine"

	// The Gorilla Web Toolkit
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	// Local Packages
	"pastebin/utils"
)

// FIXME: load auth key from elsewhere!
var CSRF = csrf.Protect([]byte("32-byte-long-auth-keyauth-key123"), csrf.Secure(false))

func init() {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/pastebin/", utils.ExtraSugar(pastebin)).Methods("GET", "POST").Name("pastebin")
	r.HandleFunc("/pastebin/about", utils.ExtraSugar(about)).Methods("GET").Name("about")

	http.Handle("/pastebin/", CSRF(r))
}

func Http404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	var tmpl = template.Must(template.ParseFiles("templates/404.tmpl"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/pastebin", http.StatusFound)
}

func about(w http.ResponseWriter, r *http.Request) {
	var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/about.tmpl"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func pastebin(w http.ResponseWriter, r *http.Request) {
	// c := appengine.NewContext(r)
	if r.Method == "GET" {
		var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl"))

		// http://www.gorillatoolkit.org/pkg/csrf
		if err := tmpl.Execute(w, map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		log.Println(r.Form)

		// FIXME: This should actually point to the actual paste
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			w.Write([]byte("/pastebin/"))
		} else {
			http.Redirect(w, r, "/pastebin/", http.StatusSeeOther)
		}
	}
}
