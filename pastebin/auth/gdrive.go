package auth

import (
	// Go Builtin Packages
	"html/template"
	"net/http"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/user"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	go_ae "google.golang.org/appengine"

	// Local Packages
	"pastebin/models"
	"pastebin/utils"
)

const response_template = `
<html>
<head>
	<title>OAuth2 Response Handler</title>
</head>
<body>
	<script>
		try {
			window.opener.HandleGAuthComplete("{{ . }}");
		} catch(e) {
			location.href = "/pastebin/";
		}
		window.close();
	</script>
</body>
</html>
`

var responseTemplate = template.Must(template.New("response").Parse(response_template))

func auth_gdrive_start(w http.ResponseWriter, r *http.Request) {
	// We need to be able to serve an inline script on this route for window.opener.*
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'")

	c := appengine.NewContext(r)
	usr := user.Current(c)
	if usr == nil { // Oops, we need a logged in user for this ^_^
		http.Redirect(w, r, "/pastebin/auth/login?next=/pastebin/auth/gdrive", http.StatusFound)
		return
	}

	// Attempt retrieving the auth token from cookie first. Yay cookies!
	havetoken, verr := models.CheckOAuthToken(c)
	utils.PanicOnErr(c, verr)

	if havetoken == true {
		err := responseTemplate.Execute(w, "success")
		utils.PanicOnErr(c, err)
		return
	}

	state_token, err := utils.SC().Encode("state-token", time.Now().Format(time.StampNano))
	utils.PanicOnErr(c, err)

	config := utils.OAuthConfigDance(c)
	authURL := config.AuthCodeURL(state_token, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func auth_gdrive_finish(w http.ResponseWriter, r *http.Request) {
	// We need to be able to serve an inline script on this route for window.opener.*
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'")

	c := appengine.NewContext(r)
	usr := user.Current(c)
	if usr == nil { // Oops, we need a logged in user for this ^_^
		http.Redirect(w, r, "/pastebin/auth/login?next=/pastebin/auth/gdrive", http.StatusFound)
		return
	}

	utils.ProcessForm(c, r)
	// Parse and validate state-token here
	var state_token string
	received_token := strings.TrimSpace(r.Form.Get("state"))
	if err := utils.SC().Decode("state-token", received_token, &state_token); err != nil {
		http.Error(w, "Oops, we couldn't validate the state token after the round trip :(", http.StatusBadRequest)
		return
	}

	// Check for errors, it's usually access_denied
	if r.Form.Get("error") == "access_denied" {
		// Make a sad face here or something -flails-
		err := responseTemplate.Execute(w, "Access denied")
		utils.PanicOnErr(c, err)
		return

	} else if r.Form.Get("error") != "" {
		err := responseTemplate.Execute(w, r.Form.Get("error"))
		utils.PanicOnErr(c, err)
		return
	}

	config := utils.OAuthConfigDance(c)

	ctx := go_ae.NewContext(r)
	code := r.Form.Get("code")
	token, err := config.Exchange(ctx, code)
	utils.PanicOnErr(c, err)

	err = models.SaveOAuthToken(c, token)
	utils.PanicOnErr(c, err)

	err = responseTemplate.Execute(w, "success")
	utils.PanicOnErr(c, err)
	return
}
