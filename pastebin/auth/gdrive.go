package auth

import (
	// Go Builtin Packages
	"html/template"
	"log"
	"net/http"

	// Google OAuth2/Drive Packages
	"google.golang.org/api/drive/v3"
)

const gDrive_response_template = `
<!DOCTYPE html>
<html lang="en">
<head>
	<title>OAuth2 Response Handler - ada-young.com</title>
</head>
<body>
	<p>This window should close on its own! Close it if it doesn't :o</p>
	<script>
		try {
			window.opener.HandleGAuthComplete("{{ . }}");
		} catch(e) {
			// TODO: handle errors if any
		}
		window.close();
	</script>
</body>
</html>
`

var gDriveResponseTemplate = template.Must(template.New("response").Parse(gDrive_response_template))

func authGDriveStart(w http.ResponseWriter, r *http.Request) {
	oauthStart(w, r, "google", "/pastebin/auth/gdrive/finish", drive.DriveFileScope)
}

func authGDriveFinish(w http.ResponseWriter, r *http.Request) {
	// We need to be able to serve an inline script on this route for window.opener.*
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'")

	var result string
	if _, err := oauthFinish(w, r, "google", drive.DriveFileScope); err == nil {
		result = "success"
	} else {
		log.Printf("ERROR: %v\n", err)
		result = err.Error()
	}

	if err := gDriveResponseTemplate.Execute(w, result); err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to say 'We dunnit!' but something went wrong.", http.StatusInternalServerError)
	}
}
