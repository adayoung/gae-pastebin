package models

import (
	// Go Builtin Packages
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/datastore"
	"appengine/user"
)

type Tags []string
type Paste struct {
	UserID  string    `datastore:"user_id"`
	Title   string    `datastore:"title"`
	Content []byte    `datastore:"content,noindex"`
	Tags    Tags      `datastore:"tags"`
	Format  string    `datastore:"format,noindex"`
	Date    time.Time `datastore:"date_published"`
	// We need the Zlib flag to correctly process old, uncompressed content
	Zlib bool `datastore:"zlib,noindex"`
}

func (p *Paste) Load(ds <-chan datastore.Property) error {
	// TODO: Do something with ErrFieldMismatch here
	if err := datastore.LoadStruct(p, ds); err != nil {
		return nil // Do nothing D:
	}
	return nil
}

func (p *Paste) Save(ds chan<- datastore.Property) error {
	return datastore.SaveStruct(p, ds)
}

const PasteDSKind string = "Paste"

func genpasteKey(c appengine.Context, p *Paste) (*datastore.Key, string) {
	timestamp := p.Date.Format(time.StampNano)

	hasher := sha256.New()
	hasher.Write([]byte(timestamp))
	hasher.Write(p.Content)
	digest := hex.EncodeToString(hasher.Sum(nil))

	paste_id := digest[:8] // This is probably a silly way to go about it xD
	return datastore.NewKey(c, PasteDSKind, paste_id, 0, nil), paste_id
}

type ValidationError struct {
	What string // What is invalid
	Why  string // Why is it invalid
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s - %s", e.What, e.Why)
}

func (p Paste) validate() error {
	// ... this looks more like a cleaner than a validator O_o

	// Title - truncate title to 50
	if len(p.Title) > 50 {
		p.Title = p.Title[:50]
	}

	// Tags - accept a maximum of 15 tags only, each of max length 15
	// Tags - must consist of alphanumeric characters only
	filter_exp := regexp.MustCompile("[^A-Za-z0-9]+")
	for index := 0; index < len(p.Tags); index++ {
		p.Tags[index] = filter_exp.ReplaceAllString(p.Tags[index], "")
		p.Tags[index] = strings.ToLower(strings.Trim(p.Tags[index], "-"))
		if len(p.Tags[index]) > 15 {
			p.Tags[index] = p.Tags[index][:15]
		}
	}
	if len(p.Tags) > 15 {
		p.Tags = p.Tags[:15]
	}

	// return &ValidationError{"Huh?", "Why?"}
	return nil
}

func (p Paste) save(c appengine.Context) (string, error) {
	if err := p.validate(); err == nil {
		key, paste_id := genpasteKey(c, &p)
		log.Printf("Creating new paste with paste_id [%s]", paste_id)
		_, err := datastore.Put(c, key, &p)
		if err != nil {
			return "", err
		}
		return paste_id, nil
	} else {
		return "", err
	}
}

func (p Paste) Delete(c appengine.Context, paste_id string) {
	key := datastore.NewKey(c, PasteDSKind, paste_id, 0, nil)
	log.Printf("Delete paste with paste_id [%s]", paste_id)
	if err := datastore.Delete(c, key); err != nil {
		log.Panic(err)
	}
}

func NewPaste(c appengine.Context, r *http.Request) (string, error) {
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

	paste.Date = time.Now()

	paste_id, err := paste.save(c)
	if err != nil {
		return "", err
	}

	return paste_id, nil
}

func GetPaste(c appengine.Context, paste_id string) (*Paste, error) {
	key := datastore.NewKey(c, PasteDSKind, paste_id, 0, nil)
	paste := new(Paste)
	if err := datastore.Get(c, key, paste); err != nil {
		return paste, err
	}
	return paste, nil
}
