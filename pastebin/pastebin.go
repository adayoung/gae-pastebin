package pastebin

import (
	// Go Builtin Packages
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"os"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/datastore"
	"appengine/user"

	// The Gorilla Web Toolkit
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	// Go Humanize by Dustin Sallings
	"github.com/dustin/go-humanize"

	// Local Packages
	api_v1 "pastebin/api/v1"
	"pastebin/auth"
	counter "pastebin/counter"
	"pastebin/models"
	"pastebin/utils"
)

func init() {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/pastebin/", utils.ExtraSugar(pastebin)).Methods("GET", "POST").Name("pastebin")
	r.HandleFunc("/pastebin/about", utils.ExtraSugar(about)).Methods("GET").Name("about")
	r.HandleFunc("/pastebin/clean", clean).Methods("GET").Name("pastecleanr") // Order is important! :o
	r.HandleFunc("/pastebin/search/", utils.ExtraSugar(search)).Methods("GET").Name("pastesearch")
	r.HandleFunc("/pastebin/{paste_id}", utils.ExtraSugar(pasteframe)).Methods("GET").Name("pasteframe")
	r.HandleFunc("/pastebin/{paste_id}/content", utils.ExtraSugar(pastecontent)).Methods("GET").Name("pastecontent")
	r.HandleFunc("/pastebin/{paste_id}/content/link", utils.ExtraSugar(pastecontent)).Methods("GET").Name("pastecontentlink")
	r.HandleFunc("/pastebin/{paste_id}/download", utils.ExtraSugar(pastecontent)).Methods("GET").Name("pastedownload")
	r.HandleFunc("/pastebin/{paste_id}/delete", utils.ExtraSugar(pastecontent)).Methods("POST").Name("pastedelete")

	r.NotFoundHandler = http.HandlerFunc(Http404)

	CSRFAuthKey := os.Getenv("CSRFAuthKey")
	CSRF := csrf.Protect([]byte(CSRFAuthKey), csrf.Secure(!appengine.IsDevAppServer()))
	http.Handle("/pastebin/", CSRF(r))

	// Here be auth handlers
	http.Handle("/pastebin/auth/", CSRF(auth.Router))

	// Here be API handlers
	http.Handle("/pastebin/api/v1/", api_v1.API_Router)
}

func Http404(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	w.WriteHeader(http.StatusNotFound)
	var tmpl = template.Must(template.ParseFiles("templates/404.tmpl"))
	if err := tmpl.Execute(w, nil); err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to make the '404' page but something went wrong.", http.StatusInternalServerError)
	}
}

func about(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/about.tmpl"))
	if err := tmpl.Execute(w, map[string]interface{}{
		"user": user.Current(c),
	}); err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to make the 'about' page but something went wrong.", http.StatusInternalServerError)
	}
}

func pastebin(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	if r.Method == "GET" {
		var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl"))

		destination := "datastore" // default destination
		if d_cookie, err := r.Cookie("dest"); err == nil {
			if d_cookie.Value == "datastore" || d_cookie.Value == "gdrive" {
				destination = d_cookie.Value
			}
		}

		// http://www.gorillatoolkit.org/pkg/csrf
		if err := tmpl.Execute(w, map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
			"user":           user.Current(c),
			"dest":           destination,
		}); err != nil {
			c.Errorf(err.Error())
			http.Error(w, "Meep! We were trying to make the 'home' page but something went wrong.", http.StatusInternalServerError)
			return
		}
	} else if r.Method == "POST" {
		paste_id, err := models.NewPaste(c, r)
		if err != nil {
			if _, ok := err.(*models.ValidationError); ok {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			c.Errorf(err.Error())
			if err, ok := err.(*models.GDriveAPIError); ok {
				http.Error(w, err.Response, err.Code)
				return
			}

			http.Error(w, "BARF!@ Something's broken!@%", http.StatusInternalServerError)
			return
		}

		if err := utils.UpdateSession(w, r, paste_id, false); err != nil {
			c.Errorf(err.Error())
			http.SetCookie(w, &http.Cookie{
				Name:     "_pb_session",
				Value:    "",
				MaxAge:   -1,
				Secure:   !appengine.IsDevAppServer(),
				HttpOnly: true,
			})
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "dest",
			Value:    r.Form.Get("destination"),
			MaxAge:   0,
			Secure:   !appengine.IsDevAppServer(),
			HttpOnly: true,
		})

		http.SetCookie(w, &http.Cookie{ // That was a HALF A KILO cookie!! :O
			Name:     "_oauth2_gdrive",
			Value:    "",
			MaxAge:   -1,
		})

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
	if len(paste_id) > 8 {
		paste_id = paste_id[:8]
	}

	if paste, err := models.GetPaste(c, paste_id); err == datastore.ErrNoSuchEntity {
		Http404(w, r)
		return
	} else if err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to retrieve this paste but something went wrong.", http.StatusInternalServerError)
		return
	} else {
		showDeleteBtn := false
		if usr != nil {
			if paste.UserID == usr.ID || user.IsAdmin(c) {
				showDeleteBtn = true
			}
		}

		if checkDelete, err := utils.CheckSession(r, paste_id); err != nil {
			c.Errorf(err.Error())
			http.Error(w, "Meep! We were trying to get a session but something went wrong.", http.StatusInternalServerError)
		} else {
			showDeleteBtn = checkDelete
		}

		defer counter.Increment(c, paste_id)
		p_count, _ := counter.Count(c, paste_id)

		var p_content string
		if paste.Format == "plain" {
			var _b_content bytes.Buffer
			_p_content := bufio.NewWriter(&_b_content)
			err := paste.ZContent(c, r, _p_content)
			if err != nil {
				c.Errorf(err.Error())
				if gerr, ok := err.(*models.GDriveAPIError); ok {
					http.Error(w, gerr.Response, gerr.Code)
				} else {
					http.Error(w, "Meep! We were trying to retrieve this paste's plain content but something went wrong.", http.StatusInternalServerError)
				}
				return
			} else {
				_p_content.Flush()
				p_content = _b_content.String()
			}
		}

		driveHosted := false
		if len(paste.GDriveID) > 0 {
			driveHosted = true
		}

		var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/paste.tmpl"))
		if err := tmpl.Execute(w, map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(r),
			"paste_id":       paste_id,
			"paste":          paste,
			"p_content":      p_content,
			"p_count":        p_count,
			"user":           usr,
			"deleteBtn":      showDeleteBtn,
			"driveHosted":    driveHosted,
		}); err != nil {
			c.Errorf(err.Error())
			http.Error(w, "Meep! We were trying to assemble this paste but something went wrong.", http.StatusInternalServerError)
			return
		}
	}
}

func pastecontent(w http.ResponseWriter, r *http.Request) {
	// This is what keeps people from abusing our pastebin ^_^
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self' 'unsafe-inline'; img-src *")

	c := appengine.NewContext(r)
	v := mux.Vars(r)
	paste_id := v["paste_id"]
	if len(paste_id) > 8 {
		paste_id = paste_id[:8]
	}

	if paste, err := models.GetPaste(c, paste_id); err == datastore.ErrNoSuchEntity {
		Http404(w, r)
		return
	} else if err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to retrieve this paste but something went wrong.", http.StatusInternalServerError)
		return
	} else {
		// Return content link alone on the /link route
		// TODO: detect 404s and remove metadata here as well
		if cl := strings.Split(r.URL.Path, "/"); cl[len(cl)-1] == "link" {
			if len(paste.GDriveDL) > 0 {
				fl_link, ferr := paste.LinkFromDrive(c, r)
				if ferr != nil {
					fl_link = ferr.Error()
					w.WriteHeader(500) // Umm.. it just needs to go to .fail() O_o
				}
				w.Write([]byte(fl_link))
				return
			} else {
				w.Write([]byte(strings.Join(strings.Split(r.URL.Path, "/")[:len(cl)-1], "/")))
				return
			}
		}

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

		// Check ownership and expire paste on the /delete route, POST is enforced here by the mux
		if dl := strings.Split(r.URL.Path, "/"); dl[len(dl)-1] == "delete" {
			canDelete := false
			if usr := user.Current(c); usr != nil {
				if paste.UserID == usr.ID || user.IsAdmin(c) {
					canDelete = true
				}
			}

			if checkDelete, err := utils.CheckSession(r, paste_id); err != nil {
				c.Errorf(err.Error())
				http.Error(w, "Meep! We were trying to get a session but something went wrong.", http.StatusInternalServerError)
			} else {
				canDelete = checkDelete
			}

			if canDelete {
				err := paste.Delete(c, r)
				if err != nil {
					if derr, ok := err.(*models.GDriveAPIError); ok {
						http.Error(w, derr.Response, derr.Code)
						return
					}
					c.Errorf(err.Error())
					http.Error(w, "Meep! We were trying to delete this paste but something went wrong.", http.StatusInternalServerError)
				}

				if err := utils.UpdateSession(w, r, paste_id, true); err != nil {
					c.Errorf(err.Error())
					http.Error(w, "Meep! We were trying to get or set a session but something went wrong. Your paste has been deleted, however!", http.StatusInternalServerError)
				}

				if r.Header.Get("X-Requested-With") == "XMLHttpRequest" { // AJAX
					w.Write([]byte("/pastebin/"))
				} else {
					// http://tools.ietf.org/html/rfc2616#section-10.3.4 / Http 303
					http.Redirect(w, r, "/pastebin/", http.StatusSeeOther)
				}
				return
			}
			return
		}

		err := paste.ZContent(c, r, w)
		if err != nil {
			c.Errorf(err.Error())
			if gerr, ok := err.(*models.GDriveAPIError); ok {
				http.Error(w, gerr.Response, gerr.Code)
			} else {
				http.Error(w, "Meep! We were trying to retrieve this paste's content but something went wrong.", http.StatusInternalServerError)
			}
			return
		}
	}
}

func clean(w http.ResponseWriter, r *http.Request) {
	// https://cloud.google.com/appengine/docs/go/config/cron#securing_urls_for_cron
	c := appengine.NewContext(r)
	sixmonthsago := time.Now().AddDate(0, 0, -180) // 180 days ago!

	old_stuff := datastore.NewQuery(models.PasteDSKind).
		Filter("date_published <", sixmonthsago).
		Filter("gdrive_id =", "").
		KeysOnly().Limit(150) // Find up to 150 old pastes
	old_keys, err := old_stuff.GetAll(c, nil)
	if err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to retrieve old pastes but something went wrong.", http.StatusInternalServerError)
		return
	}

	var paste_ids []*datastore.Key
	for _, old_key := range old_keys {
		paste_id := old_key.StringID()
		if last, _ := counter.Last(c, paste_id); sixmonthsago.After(last) == true {
			paste_ids = append(paste_ids, old_key)
		}
	}

	c.Infof("About to delete the following pastes: %s", paste_ids)
	err = datastore.DeleteMulti(c, paste_ids)
	if err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to delete old pastes but something went wrong.", http.StatusInternalServerError)
		// return // let the shards die unconditionally
	}

	// Clear counter shards here
	var shardc_dkeys []*datastore.Key
	for _, paste_id := range paste_ids {
		c_key := datastore.NewKey(c, "GeneralCounterShardConfig", paste_id.StringID(), 0, nil)
		shardc_dkeys = append(shardc_dkeys, c_key)

		shard_keys := datastore.NewQuery("GeneralCounterShard").Filter("Name =", paste_id.StringID()).KeysOnly()
		if shard_dkeys, err := shard_keys.GetAll(c, nil); err == nil {
			derr := datastore.DeleteMulti(c, shard_dkeys)
			if derr != nil {
				c.Errorf(derr.Error())
				http.Error(w, "Meep! We were trying to delete old shards but something went wrong.", http.StatusInternalServerError)
				return
			}
		} else {
			c.Errorf(err.Error())
			http.Error(w, "Meep! We were trying to retrieve old shard keys but something went wrong.", http.StatusInternalServerError)
			return
		}
	}

	err = datastore.DeleteMulti(c, shardc_dkeys)
	if err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to delete old shard keys but something went wrong.", http.StatusInternalServerError)
	}
}

func search(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	usr := user.Current(c)

	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" { // AJAX
		cursor := r.URL.Query().Get("c")
		// Let's abuse an empty Paste object to validate/clean tags
		p := new(models.Paste)
		p.Tags = strings.Split(r.URL.Query().Get("tags"), " ")
		p.Validate()

		q := datastore.NewQuery(models.PasteDSKind)
		// q = q.Project("title", "date_published", "tags") // <-- That's not allowed for '='' filter queries O_o
		for _, tag := range p.Tags {
			q = q.Filter("tags =", tag)
		}
		q = q.Order("-date_published").Limit(10)
		if len(cursor) > 0 {
			if cursor, err := datastore.DecodeCursor(cursor); err != nil {
				c.Errorf(err.Error())
				http.Error(w, "Meep! We were trying to search for pastes keys but something went wrong (Invalid Cursor?)", http.StatusInternalServerError)
				return
			} else {
				q = q.Start(cursor)
			}
		}

		var results []interface{}
		t := q.Run(c)
		for {
			q := models.Paste{}
			key, err := t.Next(&q)
			if err == datastore.Done {
				break
			}
			if err != nil {
				c.Errorf(err.Error())
				http.Error(w, "Meep! We were trying to search for pastes keys but something went wrong (Query Error)", http.StatusInternalServerError)
				break
			}

			results = append(results, map[string]interface{}{
				"paste_id": key.StringID(),
				"title":    template.HTMLEscapeString(q.Title),
				"tags":     q.Tags,
				"i_date":   q.Date.Format(time.ANSIC),
				"date":     humanize.Time(q.Date),
			})
		}

		q_result := map[string]interface{}{
			"paste": map[string]interface{}{
				"results": results,
				"tags":    p.Tags,
			},
		}

		if cursor, err := t.Cursor(); err == nil {
			q_result["cursor"] = cursor.String()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(q_result)
	} else {
		var tmpl = template.Must(template.ParseFiles("templates/base.tmpl", "pastebin/templates/pastebin.tmpl", "pastebin/templates/search.tmpl"))
		if err := tmpl.Execute(w, map[string]interface{}{
			"user": usr,
		}); err != nil {
			http.Error(w, "Meep! We were trying to make the 'search' page but something went wrong.", http.StatusInternalServerError)
		}
	}
}
