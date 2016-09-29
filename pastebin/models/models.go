package models

import (
	// Go Builtin Packages
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
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
	Date    time.Time
	Expired bool
}

const PasteDSKind string = "Paste"

func genpasteKey(c appengine.Context, p *Paste) (*datastore.Key, string) {
	timestamp := time.Now().Format(time.StampNano)

	hasher := sha256.New()
	hasher.Write([]byte(timestamp))
	hasher.Write(p.Content)
	digest := hex.EncodeToString(hasher.Sum(nil))

	paste_id := digest[:8] // This is probably a silly way to go about it xD
	return datastore.NewKey(c, PasteDSKind, paste_id, 0, nil), paste_id
}

func (p Paste) validate() error {
	// FIXME: Implement input validation here
	return nil
}

func (p Paste) save(c appengine.Context) (string, error) {
	if err := p.validate(); err == nil {
		key, stringID := genpasteKey(c, &p)
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

	paste.Tags = strings.Split(r.PostForm.Get("tags"), " ")
	paste.Format = r.PostForm.Get("format")

	if ipaddr := net.ParseIP(r.RemoteAddr); ipaddr != nil {
		paste.IPAddr = net.IP(ipaddr)
	}

	paste.Date = time.Now()
	paste.Expired = false

	stringID, _ := paste.save(c) // FIXME: do something if this returns an error
	// stringID := "meep" // DEBUG: Let's not write to the datastore at the moment :o
	return stringID
}

func GetPaste(c appengine.Context, paste_id string) (*Paste, error) {
	key := datastore.NewKey(c, PasteDSKind, paste_id, 0, nil)
	paste := new(Paste)
	if err := datastore.Get(c, key, paste); err != nil {
		return paste, err
	}
	return paste, nil
}
