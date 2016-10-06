package auth

import (
	// Go Builtin Packages
	"encoding/json"
	// "log"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	// Google Appengine Packages
	"appengine"
	"appengine/user"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	go_ae "google.golang.org/appengine"

	// Local Packages
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

func auth_gdrive_begin(w http.ResponseWriter, r *http.Request) {
	// We need to be able to serve an inline script on this route for window.opener.*
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'")

	c := appengine.NewContext(r)
	usr := user.Current(c)
	if usr == nil { // Oops, we need a logged in user for this ^_^
		http.Redirect(w, r, "/pastebin/auth/login?next=/pastebin/auth/gdrive", http.StatusFound)
		return
	}

	// Attempt retrieving the auth token from cookie first. Yay cookies!
	if cookie, err := r.Cookie("gdrive-token"); err == nil {
		value := make(map[string]string)
		if err = utils.SC().Decode("gdrive-token", cookie.Value, &value); err == nil {
			err := responseTemplate.Execute(w, "success")
			utils.PanicOnErr(c, err)
			return
		}
	}

	// Oops, no such cookie. Let's make a new one!
	GCPOAuthCID := []byte(os.Getenv("GCPOAuthCID"))
	config, err := google.ConfigFromJSON([]byte(GCPOAuthCID), drive.DriveFileScope)
	utils.PanicOnErr(c, err)

	state_token, err := utils.SC().Encode("state-token", time.Now().Format(time.StampNano))
	utils.PanicOnErr(c, err)

	authURL := config.AuthCodeURL(state_token, oauth2.AccessTypeOffline)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func auth_gdrive_complete(w http.ResponseWriter, r *http.Request) {
	// We need to be able to serve an inline script on this route for window.opener.*
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'")

	c := appengine.NewContext(r)
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

	// Do the config dance again!
	GCPOAuthCID := []byte(os.Getenv("GCPOAuthCID"))
	config, err := google.ConfigFromJSON([]byte(GCPOAuthCID), drive.DriveAppdataScope)
	utils.PanicOnErr(c, err)

	ctx := go_ae.NewContext(r)
	code := r.Form.Get("code")
	token, err := config.Exchange(ctx, code)
	utils.PanicOnErr(c, err)

	// TODO: Save teh token here and do window.opener.GDriveSuccess thingie
	lookietoken, err := json.Marshal(token)
	utils.PanicOnErr(c, err)

	yaycookie := map[string]string{
		"gdrive-token": string(lookietoken),
	}
	if encoded, err := utils.SC().Encode("gdrive-token", yaycookie); err == nil {
		cookie := &http.Cookie{
			Name:     "gdrive-token",
			Value:    encoded,
			Path:     "/pastebin/",
			MaxAge:   1.577e+7, // six months!
			Secure:   !appengine.IsDevAppServer(),
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
	}

	err = responseTemplate.Execute(w, "success")
	utils.PanicOnErr(c, err)
	return
}
