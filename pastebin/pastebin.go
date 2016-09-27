package pastebin

import (
	// Go Builtin Packages
	"html/template"
	"net/http"
	// "log"

	// Google Appengine Packages
	// "appengine"

	// The Gorilla Web Toolkit
	"github.com/gorilla/context"
	"github.com/gorilla/mux"

	// Local Packages
	"pastebin/utils"
)

func init() {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", utils.ExtraSugar(index)).Methods("GET").Name("index")
	r.HandleFunc("/pastebin", utils.ExtraSugar(pastebin)).Methods("GET", "POST").Name("pastebin")
	r.HandleFunc("/about", utils.ExtraSugar(about)).Methods("GET").Name("about")
	r.NotFoundHandler = http.HandlerFunc(Http404)

	http.Handle("/", r)
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
	var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "templates/pastebin.tmpl", "templates/about.tmpl"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func pastebin(w http.ResponseWriter, r *http.Request) {
	// c := appengine.NewContext(r)
	if r.Method == "GET" {
		var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "templates/pastebin.tmpl"))

		_xsrf_token := context.Get(r, "_xsrf_token")

		if err := tmpl.Execute(w, _xsrf_token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
