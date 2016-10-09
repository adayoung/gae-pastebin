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
	UserID string       `datastore:"user_id"`
	Token  oauth2.Token `datastore:"token,noindex"`
}

func SaveOAuthToken(c appengine.Context, t *oauth2.Token) error {
	user_id := user.Current(c).ID
	token := new(OAuthToken)
	token.UserID = user_id
	token.Token = *t
	key := datastore.NewKey(c, "OAuthToken", user_id, 0, nil)
	if _, err := datastore.Put(c, key, token); err != nil {
		return err
	}
	return nil
}

func CheckOAuthToken(c appengine.Context) (bool, error) {
	if usr := user.Current(c); usr != nil {
		if count, err := datastore.NewQuery("OAuthToken").Filter("user_id =", usr.ID).Limit(1).Count(c); err != nil {
			return false, err
		} else if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

func GetOAuthClient(c appengine.Context, r *http.Request, user_id string) (*http.Client, error) {
	key := datastore.NewKey(c, "OAuthToken", user_id, 0, nil)
	token := new(OAuthToken)
	if err := datastore.Get(c, key, token); err == nil {
		ctx := go_ae.NewContext(r)
		config, cerr := utils.OAuthConfigDance(c)
		if cerr != nil {
			return nil, cerr
		}

		client := config.Client(ctx, &token.Token) // How come this doesn't return an error? O_o

		if _, derr := datastore.Put(c, key, token); derr != nil {
			return nil, derr
		}

		return client, nil
	} else {
		return nil, err
	}

	c.Errorf("Oops, it's an error to arrive here just to return a nil client O_o")
	return nil, nil
}

func (p *Paste) saveToDrive(c appengine.Context, r *http.Request, content *bytes.Buffer, paste_id string) error {
	client, cerr := GetOAuthClient(c, r, p.UserID)
	if cerr != nil {
		return cerr
	}

	if service, err := drive.New(client); err != nil {
		return err
	} else {
		p_content := new(drive.File)
		p_content.Name = paste_id

		// Here be metadata
		appProperties := make(map[string]string)
		appProperties["PasteID"] = paste_id
		appProperties["Title"] = p.Title
		appProperties["Tags"] = strings.Join(p.Tags, ",")
		appProperties["Format"] = p.Format
		appProperties["Date"] = p.Date.Format(time.RFC3339Nano)
		appProperties["Zlib"] = fmt.Sprintf("%v", p.Zlib)

		p_content.AppProperties = appProperties
		p_content.Parents = []string{"appDataFolder"}

		buffer := bytes.NewReader(content.Bytes())
		fc_call := service.Files.Create(p_content).Fields("id")
		fc_call = fc_call.Media(buffer)
		response, err := fc_call.Do()

		if err != nil {
			c.Errorf(err.Error())
			return parseAPIError(c, err, p)
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
		// TODO: https://godoc.org/google.golang.org/api/drive/v3#FilesService.Get .. !@#
		client, cerr := GetOAuthClient(c, r, p.UserID)
		if cerr != nil {
			return cerr
		}

		if service, err := drive.New(client); err != nil {
			return err
		} else {
			fg_call := service.Files.Get(p.GDriveID)
			response, err := fg_call.Download()
			if err != nil {
				c.Errorf(err.Error())
				return parseAPIError(c, err, p)
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

				} // else {} TODO: Delete paste metadata if StatusCode == 404
			}
		}
	} else {
		return err
	}

	return nil
}
