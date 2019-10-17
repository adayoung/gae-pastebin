package models

import (
	// Go Builtin Packages
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"

	// The Gorilla Web Toolkit
	"github.com/gorilla/sessions"
	// Local Packages
	"github.com/adayoung/gae-pastebin/pastebin/utils"
)

func init() {
	gob.Register(&oauth2.Token{})
}

var sessionStore = sessions.NewCookieStore([]byte(os.Getenv("CSRFAuthKey")))

func SaveOAuthToken(w http.ResponseWriter, r *http.Request, token *oauth2.Token) error {
	if session, err := sessionStore.Get(r, "_oauth2_gdrive"); err != nil {
		return err
	} else {
		session.Options = &sessions.Options{
			Path:     "/pastebin/",
			MaxAge:   0,
			HttpOnly: true,
			Secure:   os.Getenv("CSRFSecureC") == "true",
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

func GetOAuthClient(r *http.Request, user_id string) (*http.Client, error) {
	if token, err := GetOAuthToken(r, user_id); err == nil {
		ctx := r.Context()
		config, cerr := utils.OAuthConfigDance(drive.DriveFileScope)
		if cerr != nil {
			return nil, cerr
		}

		client := config.Client(ctx, token)
		return client, nil
	} else {
		return nil, err
	}

	log.Printf("ERROR: Oops, it's an error to arrive here just to return a nil client O_o")
	return nil, nil
}

func makePastebinFolder(client *http.Client) (string, error) {
	if service, aerr := drive.New(client); aerr != nil {
		return "", aerr
	} else {
		fl_call := service.Files.List().Fields("files(id)").PageSize(1).Spaces("drive")
		fl_call = fl_call.Q("name='Pastebin!!' and trashed=false")
		response, err := fl_call.Do()

		if err != nil {
			log.Printf("ERROR: %v\n", err)
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

func (p *Paste) saveToDrive(r *http.Request, paste_id string) error {
	client, cerr := GetOAuthClient(r, p.UserID)
	if cerr != nil {
		return cerr
	}

	if service, err := drive.New(client); err != nil {
		return err
	} else {
		p_content := new(drive.File)

		if len(p.Title) > 0 {
			p_content.Name = p.Title + "__" + paste_id
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
		if pbdir_id, aerr := makePastebinFolder(client); aerr == nil {
			p_content.Parents = []string{pbdir_id}
		} else {
			return parseAPIError(r, aerr, p, false)
		}

		buffer := bytes.NewReader([]byte(p.uContent))
		fc_call := service.Files.Create(p_content).Fields("id", "webContentLink")
		fc_call = fc_call.Media(buffer)
		response, err := fc_call.Do()

		if err != nil {
			log.Printf("ERROR: %v\n", err)
			return parseAPIError(r, err, p, false)
		} else {
			log.Printf("INFO: Received Google Drive File ID -> " + response.Id)
			p.GDriveID = response.Id

			// Add a permission to allow downloading content
			fp_call := service.Permissions.Create(response.Id, &drive.Permission{
				Role: "reader", Type: "anyone",
			}).Fields("id")
			if _, p_err := fp_call.Do(); p_err != nil {
				log.Printf("ERROR: %v\n", p_err)
			} else {
				p.GDriveDL = response.WebContentLink
			}
		}
	}

	return nil
}

func (p *Paste) LinkFromDrive(r *http.Request) (string, error) {
	fl_link := "" // yay empty string

	fl_call := &http.Client{}
	fl_call.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// return http.ErrUseLastResponse // <-- FIXME: Huh? Undefined? O_o
		return fmt.Errorf("net/http: use last response")
	}
	fl_response, _ := fl_call.Head(p.GDriveDL)

	if fl_response.StatusCode == 404 {
		p.Delete()
		return "", fmt.Errorf("404 - content not found")
	}

	fl_link = fl_response.Header.Get("Location")

	if strings.Contains(fl_link, "accounts.google.com") {
		p.Delete() // Well they refuse to share it!@
		return "", fmt.Errorf("403 - content no longer shared")
	}

	return fl_link, nil
}
