package models

import (
	// Go Builtin Packages
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/datastore"
	"appengine/user"

	// Local Packages
	"pastebin/utils"
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
	// Private, uncompressed content, for validation and processing
	uContent string `datastore:"-"`
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
	// This, because Gwawr was here, and Gwawr is awesome!
	digest := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hasher.Sum(nil))

	paste_id := digest[:8] // This is probably a silly way to go about it xD
	// We're such trolls, we don't even check for collisions ^_^
	return datastore.NewKey(c, PasteDSKind, paste_id, 0, nil), paste_id
}

type ValidationError struct {
	What string // What is invalid
	Why  string // Why is it invalid
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s - %s", e.What, e.Why)
}

func (p *Paste) Validate() error {
	// ... this looks more like a cleaner than a validator O_o

	// A paste must have content!@
	if !(len(p.uContent) > 0) {
		return &ValidationError{"Content", "Oops, we need 'content' for this."}
	}

	// Title - truncate title to 50
	if len(p.Title) > 50 {
		p.Title = p.Title[:50]
	}

	// Force format to 'plain' if nothing valid is specified
	if !(p.Format == "plain" || p.Format == "html") {
		p.Format = "plain"
	}

	// Tags - accept a maximum of 15 tags only, each of max length 15
	// Tags - must consist of alphanumeric characters only
	filter_exp := regexp.MustCompile("[^A-Za-z0-9]+")
	for index := 0; index < len(p.Tags); index++ {
		p.Tags[index] = filter_exp.ReplaceAllString(p.Tags[index], "")
		p.Tags[index] = strings.ToLower(strings.TrimSpace(p.Tags[index]))
		if len(p.Tags[index]) > 15 {
			p.Tags[index] = p.Tags[index][:15]
		}
	}
	if len(p.Tags) > 15 {
		p.Tags = p.Tags[:15]
	}

	var u_tags []string
	u_tag_map := make(map[string]string)
	for _, tag := range p.Tags {
		if u_tag_map[tag] == "" {
			u_tags = append(u_tags, tag)
		}
		u_tag_map[tag] = tag
	}

	p.Tags = u_tags // Yay no more duplicates!

	// return &ValidationError{"Huh?", "Why?"}
	return nil
}

func (p *Paste) save(c appengine.Context) (string, error) {
	if err := p.Validate(); err == nil {
		// Compress content here, AFTER validation
		var content bytes.Buffer
		w := zlib.NewWriter(&content)
		w.Write([]byte(p.uContent))
		w.Close()
		p.Content = content.Bytes()
		p.Zlib = true

		key, paste_id := genpasteKey(c, p)
		c.Infof("Creating new paste with paste_id [%s]", paste_id)
		_, err := datastore.Put(c, key, p)
		if err != nil {
			return "", err
		}
		return paste_id, nil
	} else {
		return "", err
	}
}

type pasteContent interface {
	Write([]byte) (int, error)
}

func (p *Paste) ZContent(pc pasteContent) {
	if p.Zlib { // always the case with new content
		// Decompress content and write out the response
		zbuffer := bytes.NewReader(p.Content)
		ureader, _ := zlib.NewReader(zbuffer)
		io.Copy(pc, ureader)
	} else { // here be old, uncompressed content
		buffer := bytes.NewReader(p.Content)
		io.Copy(pc, buffer)
	}
}

func (p *Paste) Delete(c appengine.Context, paste_id string) {
	key := datastore.NewKey(c, PasteDSKind, paste_id, 0, nil)
	c.Infof("Delete paste with paste_id [%s]", paste_id)
	if err := datastore.Delete(c, key); err != nil {
		log.Panic(c, err)
	}
}

func NewPaste(c appengine.Context, r *http.Request) (string, error) {
	var paste Paste

	if usr := user.Current(c); usr != nil {
		paste.UserID = usr.ID
	}

	utils.ProcessForm(c, r)

	paste.Title = r.Form.Get("title")
	paste.uContent = r.Form.Get("content")

	paste.Tags = strings.Split(r.Form.Get("tags"), " ")
	paste.Format = r.Form.Get("format")
	paste.Date = time.Now()

	paste_id, err := paste.save(c)
	if err != nil {
		return "", err
	}
	// paste_id := "meep" // No saving!@

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
