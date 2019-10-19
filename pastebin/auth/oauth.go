package auth

import (
	// Go Builtin Packages
	"fmt"
	"log"
	"net/http"
	"strings"

	// Google OAuth2/Drive Packages
	"github.com/google/uuid"
	"golang.org/x/oauth2"

	// Local Packages
	"github.com/adayoung/gae-pastebin/pastebin/models"
	"github.com/adayoung/gae-pastebin/pastebin/utils"
)

func oauthStart(w http.ResponseWriter, r *http.Request, redirectPath string, scopes ...string) {
	redirectURL := "http://localhost:2019" // D:
	if r.Host != "" {
		if !strings.HasPrefix(r.Host, "localhost") {
			redirectURL = "https://" + r.Host // FIXME: We just assume we're on https
		}
	}
	redirectURL = redirectURL + redirectPath

	state_token, err := utils.SC().Encode("state-token", redirectURL)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to encode a 'state-token' but something went wrong.", http.StatusInternalServerError)
		return
	}

	if config, err := utils.OAuthConfigDance(redirectURL, scopes...); err == nil {
		// https://developers.google.com/identity/protocols/OpenIDConnect#authenticationuriparameters
		nonceStr := "-meow-"
		if nonceUID, err := uuid.NewRandom(); err == nil {
			nonceStr = nonceUID.String()
		}

		var nonce oauth2.AuthCodeOption = oauth2.SetAuthURLParam("nonce", nonceStr)
		authURL := config.AuthCodeURL(state_token, oauth2.AccessTypeOnline, nonce)
		http.Redirect(w, r, authURL, http.StatusFound)
	} else {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to do the OAuthConfigDance but something went wrong.", http.StatusInternalServerError)
	}
}

func oauthFinish(w http.ResponseWriter, r *http.Request, scopes ...string) error {
	var err error
	if err = utils.ProcessForm(r); err != nil {
		return err
	} else {
		// Check for errors, it's usually access_denied
		if r.Form.Get("error") != "" {
			return fmt.Errorf(r.Form.Get("error"))
		}
	}

	// Parse and validate state-token here
	var redirectURL string
	receivedToken := strings.TrimSpace(r.Form.Get("state"))
	if err = utils.SC().Decode("state-token", receivedToken, &redirectURL); err != nil {
		return err
	}

	config, err := utils.OAuthConfigDance(redirectURL, scopes...)
	if err != nil {
		return err
	}

	// Retrieve token in exchange for the supplied code
	ctx := r.Context()
	if token, err := config.Exchange(ctx, r.Form.Get("code")); err != nil {
		return err
	} else {
		return models.SaveOAuthToken(w, r, token)
	}
}
