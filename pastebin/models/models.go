package models

import (
	// Go Builtin Packages
	"bytes"
	"compress/gzip"
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

	"github.com/lib/pq"

	// Local Packages
	"github.com/adayoung/gae-pastebin/pastebin/cloudflare"
	"github.com/adayoung/gae-pastebin/pastebin/counter"
	"github.com/adayoung/gae-pastebin/pastebin/utils"
	"github.com/adayoung/gae-pastebin/pastebin/utils/storage"
)

type Paste struct {
	PasteID  string    `db:"paste_id"`
	UserID   string    `db:"user_id"`
	Title    string    `db:"title"`
	Content  []byte    `db:"content"`
	Tags     []string  `db:"tags"`
	Format   string    `db:"format"`
	Date     time.Time `db:"date"`
	Gzip     bool      `db:"gzip"`
	Zlib     bool      `db:"zlib"`
	uContent string    `db:"-"` // Private content, for validation and processing
	GDriveID string    `db:"gdriveid"`
	GDriveDL string    `db:"gdrivedl"`
}

func genpasteKey(p *Paste) string {
	timestamp := p.Date.Format(time.StampNano)

	maxContentHashLen := 2 * 1024 // let's read only upto the first 2kbs
	hasher := sha256.New()
	hasher.Write([]byte(timestamp))
	if len(p.Content) > maxContentHashLen {
		hasher.Write(p.Content[:maxContentHashLen])
	} else {
		hasher.Write(p.Content)
	}
	// This, because Gwawr was here, and Gwawr is awesome!
	digest := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hasher.Sum(nil))

	paste_id := digest[:8] // This is probably a silly way to go about it xD
	// We're such trolls, we don't even check for collisions ^_^
	// FIXME: check for collisions D:
	return paste_id
}

type ValidationError struct {
	What string // What is invalid
	Why  string // Why is it invalid
}

func (e *ValidationError) Error() string {
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
			if len(tag) > 0 {
				u_tags = append(u_tags, tag)
			}
		}
		u_tag_map[tag] = tag
	}

	p.Tags = u_tags // Yay no more duplicates!

	// return &ValidationError{"Huh?", "Why?"}
	return nil
}

func (p *Paste) save(r *http.Request, score float64) (string, error) {
	if err := p.Validate(); err == nil {
		// Compress content here, AFTER validation

		paste_id := genpasteKey(p)
		log.Printf("INFO: Creating new paste with paste_id [%s] [%.1f]", paste_id, score)
		p.PasteID = paste_id

		havetoken := (r.Form.Get("destination") == "gdrive")

		p.Gzip = false
		p.Zlib = false
		if havetoken == true {
			// TODO: This should should probably happen in a goroutine
			err = p.saveToDrive(r, paste_id)
			if err != nil {
				return "", err
			}
		} else {
			var content bytes.Buffer
			w := gzip.NewWriter(&content)
			w.Write([]byte(p.uContent))
			w.Close()
			p.Content = content.Bytes()
			p.Gzip = true
		}

		if err := p.saveToDB(score); err != nil {
			return "", err
		}

		return paste_id, nil
	} else {
		return "", err
	}
}

func (p *Paste) saveToDB(score float64) error {
	if len(p.Content) > (2 * 1024 * 1024) {
		return &ValidationError{"entityTooLarge", "Paste content is still over 2MB after compression."}
	}
	pasteSQL := `INSERT INTO pastebin (
			paste_id, user_id, title, content, tags,
			format, date, gzip, zlib, gdriveid, gdrivedl, rcscore
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	pasteSQL = storage.DB.Rebind(pasteSQL)
	_, err := storage.DB.Exec(pasteSQL,
		p.PasteID, p.UserID, p.Title, p.Content, pq.Array(p.Tags),
		p.Format, p.Date, p.Gzip, p.Zlib, p.GDriveID, p.GDriveDL, score,
	)
	return err
}

type pasteContent interface {
	Write([]byte) (int, error)
}

func (p *Paste) ZContent(r *http.Request, pc pasteContent) error {
	if len(p.GDriveID) > 0 {
		if err := p.loadFromDrive(r); err != nil {
			return err
		}
	}

	if !(len(p.Content) > 0) {
		if err := p.loadContent(); err != nil {
			return err
		}
	}

	buffer := bytes.NewReader(p.Content)
	if p.Zlib {
		ureader, _ := zlib.NewReader(buffer)
		defer ureader.Close()
		io.Copy(pc, ureader)
	} else if p.Format == "plain" && p.Gzip {
		ureader, _ := gzip.NewReader(buffer)
		defer ureader.Close()
		io.Copy(pc, ureader)
	} else {
		io.Copy(pc, buffer)
	}

	return nil
}

func NewPaste(r *http.Request, score float64) (string, error) {
	var paste Paste

	if err := utils.ProcessForm(r); err != nil {
		return "", err
	}

	// TODO: we do some magic with the received score here :D

	// if usr := user.Current(c); usr != nil {
	// 	paste.UserID = usr.ID
	// }

	paste.Title = r.Form.Get("title")
	paste.uContent = r.Form.Get("content")

	paste.Tags = strings.Split(r.Form.Get("tags"), " ")
	paste.Format = r.Form.Get("format")
	paste.Date = time.Now()

	paste_id, err := paste.save(r, score)
	if err != nil {
		return "", err
	}

	return paste_id, nil
}

func GetPaste(paste_id string, withContent, withTags bool) (*Paste, error) {
	var paste Paste

	query := "SELECT"
	query = query + " paste_id, user_id, title, tags,"
	if withContent {
		query = query + " content,"
	}
	query = query + " format, date, gzip, zlib, gdriveid, gdrivedl"
	query = query + " FROM pastebin WHERE paste_id=?"

	query = storage.DB.Rebind(query)
	row := storage.DB.QueryRow(query, paste_id)

	var err error
	if withContent {
		err = row.Scan(
			&paste.PasteID, &paste.UserID, &paste.Title, pq.Array(&paste.Tags),
			&paste.Content, &paste.Format, &paste.Date, &paste.Gzip,
			&paste.Zlib, &paste.GDriveID, &paste.GDriveDL,
		)
	} else {
		err = row.Scan(
			&paste.PasteID, &paste.UserID, &paste.Title, pq.Array(&paste.Tags),
			&paste.Format, &paste.Date, &paste.Gzip, &paste.Zlib,
			&paste.GDriveID, &paste.GDriveDL,
		)
	}

	return &paste, err
}

func (p *Paste) loadContent() error {
	query := "SELECT content FROM pastebin WHERE paste_id=?"
	query = storage.DB.Rebind(query)
	err := storage.DB.QueryRow(query, p.PasteID).Scan(&p.Content)
	return err
}

func (p *Paste) Delete() error {
	paste_id := p.PasteID

	query := "DELETE FROM pastebin WHERE paste_id=?"
	query = storage.DB.Rebind(query)
	if _, err := storage.DB.Exec(query, paste_id); err == nil {
		log.Printf("INFO: Delete paste with paste_id [%s]", paste_id)
	} else {
		return err
	}

	defer counter.Delete(paste_id)
	defer cloudflare.Purge(paste_id)
	return nil
}
