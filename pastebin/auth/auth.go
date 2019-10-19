package auth

import (
	// Go Builtin Packages
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	// The Gorilla Web Toolkit
	"github.com/gorilla/mux"

	// Local Packages
	"github.com/adayoung/gae-pastebin/pastebin/models"
	"github.com/adayoung/gae-pastebin/pastebin/utils"
)

func InitRoutes(r *mux.Router) {
	r.HandleFunc("/auth/gdrive/start", utils.ExtraSugar(authGDriveStart)).Methods("GET").Name("authGDriveStart")
	r.HandleFunc("/auth/gdrive/finish", utils.ExtraSugar(authGDriveFinish)).Methods("GET").Name("authGDriveFinish")

	r.HandleFunc("/auth/login", utils.ExtraSugar(authLoginStart)).Methods("GET").Name("authLoginStart")
	r.HandleFunc("/auth/google", utils.ExtraSugar(authGoogle)).Methods("GET").Name("authGoogle")
	r.HandleFunc("/auth/logout", utils.ExtraSugar(authLogout)).Methods("GET").Name("authLogout")
}

func authLoginStart(w http.ResponseWriter, r *http.Request) {
	oauthStart(w, r, "/pastebin/auth/google", "openid", "profile")
}

func authGoogle(w http.ResponseWriter, r *http.Request) {
	err := oauthFinish(w, r, "openid", "profile")
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to talk to Google but something went wrong.", http.StatusInternalServerError)
		return
	}

	if token, err := models.GetOAuthToken(r); err == nil {
		data := token.Extra("id_token")
		if data != nil {
			var idToken struct {
				UserID string `json:"sub"`
			}

			if strData, ok := data.(string); ok { // wow, golang can be awkward!
				// https://github.com/googleapis/google-auth-library-python/blob/08272d89667a901d0dff6c4ba53d5b30fcc29e81/google/auth/jwt.py#L134
				if strings.Count(strData, ".") != 2 {
					http.Error(w, "Meep! We were trying to count the dots in your token but something went wrong.", http.StatusInternalServerError)
					return
				}
				encodedData := strings.Split(strData, ".")[1]

				if jsonData, err := base64.RawURLEncoding.DecodeString(encodedData); err == nil {
					if json.Unmarshal([]byte(jsonData), &idToken); err == nil {
						if err = utils.InitAppSession(w, r, idToken.UserID); err == nil {
							utils.ClearOauthCookie(w)
							http.Redirect(w, r, "/pastebin/", http.StatusFound)
						} else {
							log.Printf("ERROR: %v\n", err)
							http.Error(w, "Meep! We were trying to initialize your session but something went wrong.", http.StatusInternalServerError)
						}
					} else {
						log.Printf("ERROR: %v\n", err)
						http.Error(w, "Meep! We were trying to parse your token but something went wrong.", http.StatusInternalServerError)
					}
				} else {
					log.Printf("ERROR: %v\n", err)
					http.Error(w, "Meep! We were trying to decode your token but something went wrong.", http.StatusInternalServerError)
				}
			} else {
				log.Printf("ERROR: %v\n", err)
				http.Error(w, "Meep! We were trying to type-assert your token but something went wrong.", http.StatusInternalServerError)
			}
		} else {
			log.Printf("WARNING: nil id_token returned\n")
			http.Error(w, "Meep! We were trying to pick your token but something went wrong.", http.StatusInternalServerError)
		}
	} else {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to fetch your token but something went wrong.", http.StatusInternalServerError)
	}
}

func authLogout(w http.ResponseWriter, r *http.Request) {
	utils.ClearAppSession(w)
	http.Redirect(w, r, "/pastebin/", http.StatusFound)
}
