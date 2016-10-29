package models

import (
	// Go Builtin Packages
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/urlfetch"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	go_ae "google.golang.org/appengine"
	"google.golang.org/appengine/memcache"

	// The Gorilla Web Toolkit
	"github.com/gorilla/sessions"

	// Local Packages
	"pastebin/utils"
)

func init() {
	gob.Register(&oauth2.Token{})
}

var sessionStore = sessions.NewCookieStore([]byte(os.Getenv("CSRFAuthKey")), []byte(os.Getenv("EncryptionK")))

func SaveOAuthToken(w http.ResponseWriter, r *http.Request, token *oauth2.Token) error {
	if session, err := sessionStore.Get(r, "_oauth2_gdrive"); err != nil {
		return err
	} else {
		session.Options = &sessions.Options{
			Path:     "/pastebin/",
			MaxAge:   0,
			HttpOnly: true,
			Secure:   !appengine.IsDevAppServer(),
		}

		session.Values["gdrive"] = token

		err = session.Save(r, w)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetOAuthToken(r *http.Request, user_id string) (*oauth2.Token, error) {
	if session, err := sessionStore.Get(r, "_oauth2_gdrive"); err != nil {
		return nil, err
	} else {
		if token, ok := session.Values["gdrive"].(*oauth2.Token); ok {
			return token, nil
		} else {
			return nil, fmt.Errorf("Invalid type for oauth2.Token.")
		}
	}
	return nil, nil
}

func GetOAuthClient(c appengine.Context, r *http.Request, user_id string) (*http.Client, error) {
	if token, err := GetOAuthToken(r, user_id); err == nil {
		ctx := go_ae.NewContext(r)
		config, cerr := utils.OAuthConfigDance(c)
		if cerr != nil {
			return nil, cerr
		}

		client := config.Client(ctx, token)
		return client, nil
	} else {
		return nil, err
	}

	c.Errorf("Oops, it's an error to arrive here just to return a nil client O_o")
	return nil, nil
}

func makePastebinFolder(c appengine.Context, client *http.Client) (string, error) {
	if service, aerr := drive.New(client); aerr != nil {
		return "", aerr
	} else {
		fl_call := service.Files.List().Fields("files(id)").PageSize(1).Spaces("drive")
		fl_call = fl_call.Q("name='Pastebin!!' and trashed=false")
		response, err := fl_call.Do()

		if err != nil {
			c.Errorf("Meep! We had an error when trying to do the FilesListCall call!")
			c.Errorf(err.Error())
			return "", err

		} else {
			if len(response.Files) > 0 { // teh Pastebin!! folder is there!
				pbdir_id := response.Files[0].Id
				return pbdir_id, nil
			} else { // teh Pastebin!! folder is NOT there!
				pbdir := new(drive.File)
				pbdir.Name = "Pastebin!!"
				pbdir.MimeType = "application/vnd.google-apps.folder"
				fc_call := service.Files.Create(pbdir).Fields("id")
				d_response, berr := fc_call.Do()
				if berr != nil {
					return "", berr
				} else {
					pbdir_id := d_response.Id
					return pbdir_id, nil
				}
			}
		}
	}
	return "", nil
}

func (p *Paste) saveToDrive(c appengine.Context, r *http.Request, paste_id string) error {
	client, cerr := GetOAuthClient(c, r, p.UserID)
	if cerr != nil {
		return cerr
	}

	if service, err := drive.New(client); err != nil {
		return err
	} else {
		p_content := new(drive.File)

		if len(p.Title) > 0 {
			p_content.Name = p.Title
		} else {
			p_content.Name = paste_id
		}

		if p.Format == "html" {
			p_content.Name = p_content.Name + ".html"
		} else {
			p_content.Name = p_content.Name + ".txt"
		}

		// Here be metadata
		appProperties := make(map[string]string)
		appProperties["PasteID"] = paste_id
		appProperties["Title"] = p.Title
		appProperties["Tags"] = strings.Join(p.Tags, ",")
		appProperties["Format"] = p.Format
		appProperties["Date"] = p.Date.Format(time.RFC3339Nano)
		appProperties["Zlib"] = fmt.Sprintf("%v", p.Zlib)

		p_content.AppProperties = appProperties
		if pbdir_id, aerr := makePastebinFolder(c, client); aerr == nil {
			p_content.Parents = []string{pbdir_id}
		} else {
			return parseAPIError(c, r, aerr, p, false)
		}

		buffer := bytes.NewReader([]byte(p.uContent))
		fc_call := service.Files.Create(p_content).Fields("id", "webContentLink")
		fc_call = fc_call.Media(buffer)
		response, err := fc_call.Do()

		if err != nil {
			c.Errorf(err.Error())
			return parseAPIError(c, r, err, p, false)
		} else {
			c.Infof("Received Google Drive File ID -> " + response.Id)
			p.GDriveID = response.Id

			// Set the thing in memcache for immediate retrieval
			mc_item := &memcache.Item{
				Key:   response.Id,
				Value: []byte(p.uContent),
			}

			ctx := go_ae.NewContext(r)
			memcache.Add(ctx, mc_item)

			// Add a permission to allow downloading content
			fp_call := service.Permissions.Create(response.Id, &drive.Permission{
				Role: "reader", Type: "anyone",
			}).Fields("id")
			if _, p_err := fp_call.Do(); p_err != nil {
				c.Errorf(p_err.Error())
			} else {
				p.GDriveDL = response.WebContentLink
			}
		}
	}

	return nil
}

func (p *Paste) LinkFromDrive(c appengine.Context, r *http.Request) (string, error) {
	fl_link := "" // yay empty string
	ctx := go_ae.NewContext(r)
	if item, err := memcache.Get(ctx, fmt.Sprintf("fl_%s", p.PasteID)); err == nil {
		fl_link = string(item.Value)
	} else if err == memcache.ErrCacheMiss {
		fl_call := urlfetch.Client(c)
		fl_call.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			// return http.ErrUseLastResponse // <-- FIXME: Huh? Undefined? O_o
			return errors.New("net/http: use last response")
		}
		fl_response, _ := fl_call.Head(p.GDriveDL)

		if fl_response.StatusCode == 404 {
			p.Delete(c, r)
			return "", errors.New("404 - content not found")
		}

		fl_link = fl_response.Header.Get("Location")

		if strings.Contains(fl_link, "accounts.google.com") {
			p.Delete(c, r) // Well they refuse to share it!@
			return "", errors.New("403 - content no longer shared")
		}

		mc_item := &memcache.Item{
			Key:   fmt.Sprintf("fl_%s", p.PasteID),
			Value: []byte(fl_link),
		}
		memcache.Add(ctx, mc_item)
	}
	return fl_link, nil
}

func (p *Paste) loadFromDrive(c appengine.Context, r *http.Request) error {
	ctx := go_ae.NewContext(r)
	if item, err := memcache.Get(ctx, p.GDriveID); err == nil {
		p.Content = item.Value
		return nil
	} else if err == memcache.ErrCacheMiss {
		fl_link, ferr := p.LinkFromDrive(c, r);
		if ferr != nil {
			return ferr
		}

		fg_call := urlfetch.Client(c)
		if response, err := fg_call.Get(fl_link); err != nil {
			c.Errorf(err.Error())
			return parseAPIError(c, r, err, p, false)

		} else {
			if response.StatusCode == 200 {
				if p_content, err := ioutil.ReadAll(response.Body); err == nil {
					p.Content = p_content

					// Set the thing in memcache for immediate retrieval
					mc_item := &memcache.Item{
						Key:   p.GDriveID,
						Value: p_content,
					}

					ctx := go_ae.NewContext(r)
					memcache.Add(ctx, mc_item)

				} else {
					return err
				}
			} else if response.StatusCode == 404 {
				p.Delete(c, r) // No err here, we just want to get rid of it xD
				return &GDriveAPIError{
					Code: 404,
					Response: "404 - content not found",
				}
			}
		}
	} else {
		return err
	}

	return nil
}
