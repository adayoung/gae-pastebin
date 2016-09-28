package models

import (
	// Go Builtin Packages
	"bytes"
	"compress/zlib"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/datastore"
	"appengine/user"
)

type Tags []string
type Paste struct {
	UserID  string
	Title   string
	Content []byte
	Tags    Tags
	Format  string
	IPAddr  net.IP
	Zlib    bool
	Date    time.Time
	Expired bool
}

const PasteDSKind string = "Paste"

func pasteKey(c appengine.Context) (*datastore.Key, string) {
	x := "xyz" // FIXME: Generate unique keys here
	return datastore.NewKey(c, PasteDSKind, x, 0, nil), x
}

func (p Paste) validate() error {
	// FIXME: Implement input validation here
	return nil
}

func (p Paste) save(c appengine.Context) (string, error) {
	if err := p.validate(); err == nil {
		key, stringID := pasteKey(c)
		_, err := datastore.Put(c, key, &p)
		if err != nil {
			log.Panicln(err)
		}
		return stringID, nil
	} else {
		return "", err
	}
}

func NewPaste(c appengine.Context, r *http.Request) string {
	var paste Paste

	if usr := user.Current(c); usr != nil {
		paste.UserID = usr.ID
	}

	paste.Title = r.PostForm.Get("title")

	var content bytes.Buffer
	w := zlib.NewWriter(&content)
	w.Write([]byte(r.PostForm.Get("content")))
	w.Close()
	paste.Content = content.Bytes()
	paste.Zlib = true

	paste.Tags = strings.Split(r.PostForm.Get("tags"), " ")
	paste.Format = r.PostForm.Get("format")

	if ipaddr := net.ParseIP(r.RemoteAddr); ipaddr != nil {
		paste.IPAddr = net.IP(ipaddr)
	}

	paste.Date = time.Now()
	paste.Expired = false

	stringID, _ := paste.save(c) // FIXME: do something if this returns an error
	return stringID
}
