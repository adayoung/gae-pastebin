package pastebin

import (
	// Go Builtin Packages
	"html/template"
	"net/http"
	"compress/zlib"
	"bytes"
	"bufio"
	"io"
	"log"

	// Google Appengine Packages
	"appengine"
	"appengine/datastore"
	"appengine/user"

	// The Gorilla Web Toolkit
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	// Local Packages
	"pastebin/auth"
	"pastebin/models"
	"pastebin/utils"
)

// FIXME: load auth key from elsewhere!
var CSRF = csrf.Protect([]byte("32-byte-long-auth-keyauth-key123"), csrf.Secure(false))

func init() {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/pastebin/about", utils.ExtraSugar(about)).Methods("GET").Name("about")
	r.HandleFunc("/pastebin/login", auth.Login).Methods("GET").Name("login")
	r.HandleFunc("/pastebin/logout", auth.Logout).Methods("GET").Name("logout")

	r.HandleFunc("/pastebin/", utils.ExtraSugar(pastebin)).Methods("GET", "POST").Name("pastebin")
	r.HandleFunc("/pastebin/{paste_id}", utils.ExtraSugar(pasteframe)).Methods("GET").Name("pasteframe")
	r.HandleFunc("/pastebin/{paste_id}/content", utils.ExtraSugar(pastecontent)).Methods("GET").Name("pastecontent")
	// r.HandleFunc("/pastebin/{paste_id}/download", utils.ExtraSugar(pastedownload)).Methods("GET").Name("pastedownload")
	// r.HandleFunc("/pastebin/{paste_id}/delete", pastedelete).Methods("POST").Name("pastedelete")

	r.NotFoundHandler = http.HandlerFunc(Http404)

	http.Handle("/pastebin/", CSRF(r))
}

func Http404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	var tmpl = template.Must(template.ParseFiles("templates/404.tmpl"))
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func about(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/about.tmpl"))
	if err := tmpl.Execute(w, map[string]interface{}{
		"user": user.Current(c),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func pastebin(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	if r.Method == "GET" {
		var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl"))

		// http://www.gorillatoolkit.org/pkg/csrf
		if err := tmpl.Execute(w, map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
			"user":           user.Current(c),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		paste_id := models.NewPaste(c, r)

		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" { // AJAX
			w.Write([]byte("/pastebin/" + paste_id))
		} else {
			// http://tools.ietf.org/html/rfc2616#section-10.3.4 / Http 303
			http.Redirect(w, r, "/pastebin/"+paste_id, http.StatusSeeOther)
		}
	}
}

func pasteframe(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	usr := user.Current(c)
	v := mux.Vars(r)
	paste_id := v["paste_id"]

	if paste, err := models.GetPaste(c, paste_id); err == datastore.ErrNoSuchEntity {
		Http404(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		showDeleteBtn := false
		if usr != nil {
			if paste.UserID == usr.ID || user.IsAdmin(c) {
				showDeleteBtn = true
			}
		}

		// TODO: Decompress paste.Content if paste.Zlib is true and paste.format is 'plain' here

		var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/paste.tmpl"))
		if err := tmpl.Execute(w, map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
			"paste_id":       paste_id,
			"paste":          paste,
			"user":           usr,
			"deleteBtn":      showDeleteBtn,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func pastecontent(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	v := mux.Vars(r)
	paste_id := v["paste_id"]

	if paste, err := models.GetPaste(c, paste_id); err == datastore.ErrNoSuchEntity {
		Http404(w, r)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		if paste.Zlib == true {
			zbuffer := bytes.NewReader(paste.Content)
			ureader, err := zlib.NewReader(zbuffer)
			if err != nil {
				log.Panic(err)
			}
			io.Copy(w, ureader)
		} else {
			w.Write(paste.Content)
		}
	}
}
