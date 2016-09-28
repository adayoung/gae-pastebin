package gae_pastebin

import (
	// Go Builtin Packages
	"html/template"
	"net/http"

	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"
)

func init() {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/", index).Methods("GET").Name("index")
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
