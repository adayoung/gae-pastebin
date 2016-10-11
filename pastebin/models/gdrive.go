package models

import (
	// Go Builtin Packages
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/datastore"
	"appengine/user"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	go_ae "google.golang.org/appengine"
	"google.golang.org/appengine/memcache"

	// Local Packages
	"pastebin/utils"
)

type OAuthToken struct {
	UserID  string       `datastore:"user_id"`
	Token   oauth2.Token `datastore:"token,noindex"`
	BatchID string       `datastore:"batch_id,noindex"`
}

func SaveOAuthToken(c appengine.Context, t *oauth2.Token) error {
	user_id := user.Current(c).ID
	token := new(OAuthToken)
	token.UserID = user_id
	token.Token = *t
	token.BatchID = fmt.Sprintf("%s_%s", user_id, time.Now().Format(time.RFC3339Nano))

	key := datastore.NewKey(c, "OAuthToken", user_id, 0, nil)
	if _, err := datastore.Put(c, key, token); err != nil {
		return err
	}
	return nil
}

func DeleteOAuthToken(c appengine.Context, batch_id string, user_id string) error {
	if berr := NewBatchCleanQ(c, batch_id); berr != nil {
		return berr
	}

	key := datastore.NewKey(c, "OAuthToken", user_id, 0, nil)
	err := datastore.Delete(c, key)
	return err
}

func CheckOAuthToken(c appengine.Context) (bool, error) {
	if usr := user.Current(c); usr != nil {
		if count, err := datastore.NewQuery("OAuthToken").Filter("user_id =", usr.ID).KeysOnly().Limit(1).Count(c); err != nil {
			return false, err
		} else if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

func GetOAuthToken(c appengine.Context, user_id string) (*OAuthToken, *datastore.Key, error) {
	key := datastore.NewKey(c, "OAuthToken", user_id, 0, nil)
	token := new(OAuthToken)
	err := datastore.Get(c, key, token)
	return token, key, err
}

func GetOAuthClient(c appengine.Context, r *http.Request, user_id string) (*http.Client, string, error) {
	if token, key, err := GetOAuthToken(c, user_id); err == nil {
		ctx := go_ae.NewContext(r)
		config, cerr := utils.OAuthConfigDance(c)
		if cerr != nil {
			return nil, "", cerr
		}

		t_source := config.TokenSource(ctx, &token.Token)
		client := oauth2.NewClient(ctx, t_source)
		n_token, terr := t_source.Token()
		if terr != nil { // terrrr!!
			return nil, "", terr
		}
		token.Token = *n_token

		if _, derr := datastore.Put(c, key, token); derr != nil {
			return nil, "", derr
		}

		return client, token.BatchID, nil
	} else {
		return nil, "", err
	}

	c.Errorf("Oops, it's an error to arrive here just to return a nil client O_o")
	return nil, "", nil
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

func (p *Paste) saveToDrive(c appengine.Context, r *http.Request, content *bytes.Buffer, paste_id string) error {
	client, batch_id, cerr := GetOAuthClient(c, r, p.UserID)
	if cerr != nil {
		return cerr
	}

	p.BatchID = batch_id
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

		p_content.Name = p_content.Name + ".zl"

		// Here be metadata
		appProperties := make(map[string]string)
		appProperties["PasteID"] = paste_id
		appProperties["Title"] = p.Title
		appProperties["Tags"] = strings.Join(p.Tags, ",")
		appProperties["Format"] = p.Format
		appProperties["Date"] = p.Date.Format(time.RFC3339Nano)
		appProperties["Zlib"] = fmt.Sprintf("%v", p.Zlib)
		appProperties["BatchID"] = p.BatchID

		p_content.AppProperties = appProperties
		if pbdir_id, aerr := makePastebinFolder(c, client); aerr == nil {
			p_content.Parents = []string{pbdir_id}
		} else {
			return parseAPIError(c, r, aerr, p, false)
		}

		buffer := bytes.NewReader(content.Bytes())
		fc_call := service.Files.Create(p_content).Fields("id")
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
				Value: content.Bytes(),
			}

			ctx := go_ae.NewContext(r)
			memcache.Add(ctx, mc_item)
		}
	}

	return nil
}

func (p *Paste) loadFromDrive(c appengine.Context, r *http.Request) error {
	ctx := go_ae.NewContext(r)
	if item, err := memcache.Get(ctx, p.GDriveID); err == nil {
		p.Content = item.Value
		return nil
	} else if err == memcache.ErrCacheMiss {
		client, _, cerr := GetOAuthClient(c, r, p.UserID)
		if cerr != nil {
			return cerr
		}

		if service, err := drive.New(client); err != nil {
			return err
		} else {
			fg_call := service.Files.Get(p.GDriveID)
			// We should probably issue a Do() call first to find out whether the file was trashed before
			// downloading it. Oorrrr... we could just tell the users to delete it from trash as well! xD
			response, err := fg_call.Download()
			if err != nil {
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

				}
			}
		}
	} else {
		return err
	}

	return nil
}

func (p *Paste) deleteFromDrive(c appengine.Context, r *http.Request) error {
	client, _, cerr := GetOAuthClient(c, r, p.UserID)
	if cerr != nil {
		return cerr
	}

	if service, err := drive.New(client); err != nil {
		return err
	} else {
		fd_call := service.Files.Delete(p.GDriveID)
		err := fd_call.Do()
		if err != nil {
			c.Errorf(err.Error())
			return parseAPIError(c, r, err, p, true)
		}
	}

	return nil
}
