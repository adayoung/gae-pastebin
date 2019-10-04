package auth

import (
	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"

	// Local Packages
	"github.com/adayoung/gae-pastebin/pastebin/utils"
)

func InitRoutes(r *mux.Router) {
	r.HandleFunc("/auth/gdrive/start", utils.ExtraSugar(auth_gdrive_start)).Methods("GET").Name("auth_gdrive_start")
	r.HandleFunc("/auth/gdrive/finish", utils.ExtraSugar(auth_gdrive_finish)).Methods("GET").Name("auth_gdrive_finish")
}
