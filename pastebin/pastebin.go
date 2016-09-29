package pastebin

import (
	// Go Builtin Packages
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"

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
	r.HandleFunc("/pastebin/{paste_id}/download", utils.ExtraSugar(pastecontent)).Methods("GET").Name("pastedownload")
	r.HandleFunc("/pastebin/{paste_id}/delete", utils.ExtraSugar(pastecontent)).Methods("POST").Name("pastedelete")

	r.NotFoundHandler = http.HandlerFunc(Http404)

	http.Handle("/pastebin/", CSRF(r))
}

func Http404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	var tmpl = template.Must(template.ParseFiles("templates/404.tmpl"))
	if err := tmpl.Execute(w, nil); err != nil {
		log.Panic(err)
	}
}

func about(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/about.tmpl"))
	if err := tmpl.Execute(w, map[string]interface{}{
		"user": user.Current(c),
	}); err != nil {
		log.Panic(err)
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
			log.Panic(err)
		}
	} else if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			log.Panic(err)
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
		log.Panic(err)
	} else {
		showDeleteBtn := false
		if usr != nil {
			if paste.UserID == usr.ID || user.IsAdmin(c) {
				showDeleteBtn = true
			}
		}

		var p_content bytes.Buffer
		if paste.Format == "plain" {
			_p_content := bufio.NewWriter(&p_content)
			if paste.Zlib {
				zbuffer := bytes.NewReader(paste.Content)
				ureader, err := zlib.NewReader(zbuffer)
				if err != nil {
					log.Panic(err)
				}

				io.Copy(_p_content, ureader)
			} else {
				buffer := bytes.NewReader(paste.Content)
				io.Copy(_p_content, buffer)
			}
		}

		var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/paste.tmpl"))
		if err := tmpl.Execute(w, map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
			"paste_id":       paste_id,
			"paste":          paste,
			"p_content":      p_content.String(),
			"user":           usr,
			"deleteBtn":      showDeleteBtn,
		}); err != nil {
			log.Panic(err)
		}
	}
}

func pastecontent(w http.ResponseWriter, r *http.Request) {
	// This is what keeps people from abusing our pastebin ^_^
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self' 'unsafe-inline'; img-src *")

	c := appengine.NewContext(r)
	v := mux.Vars(r)
	paste_id := v["paste_id"]

	if paste, err := models.GetPaste(c, paste_id); err == datastore.ErrNoSuchEntity {
		Http404(w, r)
		return
	} else if err != nil {
		log.Panic(err)
	} else {
		// Add a Content-Disposition header on the /download route
		if dl := strings.Split(r.URL.Path, "/"); dl[len(dl)-1] == "download" {
			var p_title, p_extn, dl_disposition string

			if len(paste.Title) > 0 {
				p_title = paste.Title
			} else {
				p_title = paste_id
			}

			if paste.Format == "html" {
				p_extn = "html"
			} else {
				p_extn = "txt"
			}

			dl_disposition = fmt.Sprintf("attachment; filename=\"%s.%s\"", p_title, p_extn)
			w.Header().Set("Content-Disposition", dl_disposition)
		}

		// Check ownership and expire paste on the /delete route
		if dl := strings.Split(r.URL.Path, "/"); dl[len(dl)-1] == "delete" {
			if usr := user.Current(c); usr != nil {
				if paste.UserID == usr.ID || user.IsAdmin(c) {
					paste.Delete(c, paste_id)
					if r.Header.Get("X-Requested-With") == "XMLHttpRequest" { // AJAX
						w.Write([]byte("/pastebin/"))
					} else {
						// http://tools.ietf.org/html/rfc2616#section-10.3.4 / Http 303
						http.Redirect(w, r, "/pastebin/", http.StatusSeeOther)
					}
					return
				}
			}
		}

		if paste.Zlib {
			// Decompress content and write out the response
			zbuffer := bytes.NewReader(paste.Content)
			ureader, err := zlib.NewReader(zbuffer)
			if err != nil {
				log.Panic(err)
			}

			io.Copy(w, ureader)
		} else {
			buffer := bytes.NewReader(paste.Content)
			io.Copy(w, buffer)
		}
	}
}
