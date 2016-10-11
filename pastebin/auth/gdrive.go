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
		http.Redirect(w, r, "/pastebin/auth/login?next=/pastebin/auth/gdrive/start", http.StatusFound)
		return
	}

	state_token, err := utils.SC().Encode("state-token", time.Now().Format(time.StampNano))
	if err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to encode a 'state-token' but something went wrong.", http.StatusInternalServerError)
		return
	}

	if config, err := utils.OAuthConfigDance(c); err == nil {
		authURL := config.AuthCodeURL(state_token, oauth2.AccessTypeOffline)
		http.Redirect(w, r, authURL, http.StatusFound)
	} else {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to do the OAuthConfigDance but something went wrong.", http.StatusInternalServerError)
	}
}

func auth_gdrive_finish(w http.ResponseWriter, r *http.Request) {
	// We need to be able to serve an inline script on this route for window.opener.*
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'")

	c := appengine.NewContext(r)
	usr := user.Current(c)
	if usr == nil { // Oops, we need a logged in user for this ^_^
		http.Redirect(w, r, "/pastebin/auth/login?next=/pastebin/auth/gdrive/start", http.StatusFound)
		return
	}

	if err := utils.ProcessForm(c, r); err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to process an input but something went wrong.", http.StatusInternalServerError)
		return
	}

	// Parse and validate state-token here
	var state_token string
	received_token := strings.TrimSpace(r.Form.Get("state"))
	if err := utils.SC().Decode("state-token", received_token, &state_token); err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Oops, we couldn't validate the state token after the round trip :(", http.StatusBadRequest)
		return
	}

	// Check for errors, it's usually access_denied
	if r.Form.Get("error") == "access_denied" {
		// Make a sad face here or something -flails-
		if err := responseTemplate.Execute(w, "Meep! Access Denied!"); err != nil {
			c.Errorf(err.Error())
			http.Error(w, "Meep! We were trying to say 'Access Denied' but something went wrong.", http.StatusInternalServerError)
			return
		}
		return

	} else if r.Form.Get("error") != "" {
		if err := responseTemplate.Execute(w, r.Form.Get("error")); err != nil {
			c.Errorf(err.Error())
			http.Error(w, "Meep! We were trying to say 'Access Denied' but something went wrong.", http.StatusInternalServerError)
			return
		}
		return
	}

	config, cerr := utils.OAuthConfigDance(c)
	if cerr != nil {
		c.Errorf(cerr.Error())
		http.Error(w, "Meep! We were trying to do the OAuthConfigDance but something went wrong.", http.StatusInternalServerError)
		return
	}

	ctx := go_ae.NewContext(r)
	code := r.Form.Get("code")
	token, err := config.Exchange(ctx, code)
	if err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to exchange the auth code for a token but something went wrong.", http.StatusInternalServerError)
		return
	}

	client := config.Client(ctx, token)
	err = models.SaveOAuthToken(c, client, token)
	if err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to save the OAuth Token but something went wrong.", http.StatusInternalServerError)
		return
	}

	if err = responseTemplate.Execute(w, "success"); err != nil {
		c.Errorf(err.Error())
		http.Error(w, "Meep! We were trying to say 'We dunnit!' but something went wrong.", http.StatusInternalServerError)
	}
}
