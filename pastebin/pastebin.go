package pastebin

import (
	// Go Builtin Packages
	"html/template"
	"net/http"

	// Google Appengine Packages
	// "appengine"

	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"

	// Local Packages
	"pastebin/utils"
)

func init() {
	r := mux.NewRouter()
	r.HandleFunc("/", utils.ExtraSugar(index)).Name("index")
	r.HandleFunc("/pastebin/", utils.ExtraSugar(pastebin)).Name("pastebin")

	http.Handle("/", r)
}

func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/pastebin/", http.StatusFound)
}

func pastebin(w http.ResponseWriter, r *http.Request) {
	// c := appengine.NewContext(r)
	var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "templates/pastebin.tmpl"))

	if err := tmpl.Execute(w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
