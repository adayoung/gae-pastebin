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
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/datastore"
	"appengine/user"

	// The Gorilla Web Toolkit
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	// Local Packages
	"pastebin/auth"
	counter "pastebin/counter"
	"pastebin/models"
	"pastebin/utils"
)

// FIXME: load auth key from elsewhere!
const csrf_auth_key string = "u<:>ZoTv3d<45!Bn?ionCt4*4&t;SpV;"

func init() {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/pastebin/about", utils.ExtraSugar(about)).Methods("GET").Name("about")
	r.HandleFunc("/pastebin/login", auth.Login).Methods("GET").Name("login")
	r.HandleFunc("/pastebin/logout", auth.Logout).Methods("GET").Name("logout")

	r.HandleFunc("/pastebin/", utils.ExtraSugar(pastebin)).Methods("GET", "POST").Name("pastebin")
	r.HandleFunc("/pastebin/clean", clean).Methods("GET").Name("pastecleanr") // Order is important! :o
	r.HandleFunc("/pastebin/{paste_id}", utils.ExtraSugar(pasteframe)).Methods("GET").Name("pasteframe")
	r.HandleFunc("/pastebin/{paste_id}/content", utils.ExtraSugar(pastecontent)).Methods("GET").Name("pastecontent")
	r.HandleFunc("/pastebin/{paste_id}/download", utils.ExtraSugar(pastecontent)).Methods("GET").Name("pastedownload")
	r.HandleFunc("/pastebin/{paste_id}/delete", utils.ExtraSugar(pastecontent)).Methods("POST").Name("pastedelete")

	r.NotFoundHandler = http.HandlerFunc(Http404)

	CSRF := csrf.Protect([]byte(csrf_auth_key), csrf.Secure(true))
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

		paste_id, err := models.NewPaste(c, r)
		if err != nil {
			if _, ok := err.(models.ValidationError); !ok {
				http.Error(w, err.Error(), 400)
				return
			} else {
				http.Error(w, "BARF!@ Something's broken!@%", 500)
				return
			}
		}

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

		counter.Increment(c, paste_id)
		p_count, _ := counter.Count(c, paste_id)

		var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/paste.tmpl"))
		if err := tmpl.Execute(w, map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
			"paste_id":       paste_id,
			"paste":          paste,
			"p_content":      p_content.String(),
			"p_count":        p_count,
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

func clean(w http.ResponseWriter, r *http.Request) {
	// https://cloud.google.com/appengine/docs/go/config/cron#securing_urls_for_cron
	if r.Header.Get("X-Appengine-Cron") != "true" {
		http.Error(w, "The /clean route is avaiable to cron job only.", http.StatusForbidden)
	}

	c := appengine.NewContext(r)
	threemonthsago := time.Now().AddDate(0, 0, -90) // 3 months/90 days ago

	old_stuff := datastore.NewQuery(models.PasteDSKind).
		Filter("date_published <", threemonthsago).
		KeysOnly().Limit(150) // Find up to 150 old pastes
	old_keys, err := old_stuff.GetAll(c, nil)
	if err != nil {
		log.Panic(err)
	}

	var paste_ids []*datastore.Key
	for _, old_key := range old_keys {
		paste_id := old_key.StringID()
		if last, _ := counter.Last(c, paste_id); threemonthsago.After(last) == true {
			paste_ids = append(paste_ids, old_key)
		}
	}

	log.Printf("About to delete the following pastes: %s", paste_ids)
	if err := datastore.DeleteMulti(c, paste_ids); err != nil {
		log.Panic(err)
	}

	// Clear counter shards here
	var shardc_dkeys []*datastore.Key
	for _, paste_id := range paste_ids {
		c_key := datastore.NewKey(c, "GeneralCounterShardConfig", paste_id.StringID(), 0, nil)
		shardc_dkeys = append(shardc_dkeys, c_key)

		shard_keys := datastore.NewQuery("GeneralCounterShard").Filter("Name =", paste_id.StringID()).KeysOnly()
		if shard_dkeys, err := shard_keys.GetAll(c, nil); err == nil {
			if derr := datastore.DeleteMulti(c, shard_dkeys); derr != nil {
				log.Panic(derr)
			}
		} else {
			log.Panic(err)
		}
	}

	if err := datastore.DeleteMulti(c, shardc_dkeys); err != nil {
		log.Panic(err)
	}
}
