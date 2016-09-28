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

func NewPaste(c appengine.Context, r *http.Request) string {
	var paste Paste
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

	key, stringID := pasteKey(c)
	_, err := datastore.Put(c, key, &paste)
	if err != nil {
		log.Fatal(err.Error())
	}

	return stringID
}
