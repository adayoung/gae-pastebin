package pastebin

import (
	// Go Builtin Packages
	"html/template"
	"net/http"

	// Google Appengine Packages
	// "appengine"

	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"
	"github.com/gorilla/context"

	// Local Packages
	"pastebin/utils"
)

func init() {
	r := mux.NewRouter()
	r.HandleFunc("/", utils.ExtraSugar(index)).Name("index")
	r.HandleFunc("/pastebin/", utils.ExtraSugar(pastebin)).Name("pastebin")
	r.NotFoundHandler = http.HandlerFunc(Http404)

	http.Handle("/", r)
}

func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/pastebin/", http.StatusFound)
}

func Http404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	var tmpl = template.Must(template.ParseFiles("templates/404.tmpl"))
	if err := tmpl.Execute(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func pastebin(w http.ResponseWriter, r *http.Request) {
	// c := appengine.NewContext(r)
	var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "templates/pastebin.tmpl"))

	_xsrf_token := context.Get(r, "_xsrf_token")

	if err := tmpl.Execute(w, _xsrf_token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
