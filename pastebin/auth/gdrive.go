package auth

import (
	// Go Builtin Packages
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	// Google OAuth2/Drive Packages
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"

	// Local Packages
	"github.com/adayoung/gae-pastebin/pastebin/models"
	"github.com/adayoung/gae-pastebin/pastebin/utils"
)

const response_template = `
<html>
<head>
	<title>OAuth2 Response Handler</title>
</head>
<body>
	<p>This window should close on its own! Close it if it doesn't :o</p>
	<script>
		try {
			window.opener.HandleGAuthComplete("{{ . }}");
		} catch(e) {}
		window.close();
	</script>
</body>
</html>
`

var responseTemplate = template.Must(template.New("response").Parse(response_template))

func authGDriveStart(w http.ResponseWriter, r *http.Request) {
	// We need to be able to serve an inline script on this route for window.opener.*
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'")

	state_token, err := utils.SC().Encode("state-token", time.Now().Format(time.StampNano))
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to encode a 'state-token' but something went wrong.", http.StatusInternalServerError)
		return
	}

	if config, err := utils.OAuthConfigDance(drive.DriveFileScope); err == nil {
		authURL := config.AuthCodeURL(state_token, oauth2.AccessTypeOnline)
		http.Redirect(w, r, authURL, http.StatusFound)
	} else {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to do the OAuthConfigDance but something went wrong.", http.StatusInternalServerError)
	}
}

func authGDriveFinish(w http.ResponseWriter, r *http.Request) {
	// We need to be able to serve an inline script on this route for window.opener.*
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'")

	if err := utils.ProcessForm(r); err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to process an input but something went wrong.", http.StatusInternalServerError)
		return
	}

	// Parse and validate state-token here
	var state_token string
	received_token := strings.TrimSpace(r.Form.Get("state"))
	if err := utils.SC().Decode("state-token", received_token, &state_token); err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Oops, we couldn't validate the state token after the round trip :(", http.StatusBadRequest)
		return
	}

	// Check for errors, it's usually access_denied
	if r.Form.Get("error") == "access_denied" {
		// Make a sad face here or something -flails-
		if err := responseTemplate.Execute(w, "Meep! Access Denied!"); err != nil {
			log.Printf("ERROR: %v\n", err)
			http.Error(w, "Meep! We were trying to say 'Access Denied' but something went wrong.", http.StatusInternalServerError)
			return
		}
		return

	} else if r.Form.Get("error") != "" {
		if err := responseTemplate.Execute(w, r.Form.Get("error")); err != nil {
			log.Printf("ERROR: %v\n", err)
			http.Error(w, "Meep! We were trying to say 'Access Denied' but something went wrong.", http.StatusInternalServerError)
			return
		}
		return
	}

	config, cerr := utils.OAuthConfigDance(drive.DriveFileScope)
	if cerr != nil {
		log.Printf("ERROR: %v\n", cerr)
		http.Error(w, "Meep! We were trying to do the OAuthConfigDance but something went wrong.", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	code := r.Form.Get("code")
	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to exchange the auth code for a token but something went wrong.", http.StatusInternalServerError)
		return
	}

	err = models.SaveOAuthToken(w, r, token)
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to save the OAuth Token but something went wrong.", http.StatusInternalServerError)
		return
	}

	if err = responseTemplate.Execute(w, "success"); err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to say 'We dunnit!' but something went wrong.", http.StatusInternalServerError)
	}
}
