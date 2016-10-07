package models

import (
	// Go Builtin Packages
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	// Google Appengine Packages
	"appengine"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	go_ae "google.golang.org/appengine"
	"google.golang.org/appengine/memcache"

	// Local Packages
	"pastebin/utils"
)

func (p *Paste) saveToDrive(c appengine.Context, r *http.Request, content *bytes.Buffer, paste_id string, t *interface{}) error {
	token := &oauth2.Token{}
	lolwhat, _ := json.Marshal(t)
	err := json.Unmarshal(lolwhat, &token)
	if err != nil {
		return err
	}

	ctx := go_ae.NewContext(r)
	config := utils.OAuthConfigDance(c)
	client := config.Client(ctx, token)

	if service, err := drive.New(client); err != nil {
		log.Panic(c, err)
	} else {
		p_content := new(drive.File)
		p_content.Name = paste_id
		p_content.Parents = []string{"appDataFolder"}

		buffer := bytes.NewReader(content.Bytes())
		fc_call := service.Files.Create(p_content).Fields("id")
		fc_call = fc_call.Media(buffer)
		response, err := fc_call.Do()

		if err != nil {
			c.Errorf(err.Error())
			return err
		} else {
			c.Infof("Received Google Drive File ID -> " + response.Id)
			p.GDriveID = response.Id

			// Set the thing in memcache for immediate retrieval
			mc_item := &memcache.Item{
				Key:   response.Id,
				Value: content.Bytes(),
			}
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
	} else {
		log.Panic(c, err)
	}

	return nil
}
