package auth

import (
	// Go Builtin Packages
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
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

	r.HandleFunc("/auth/github/start", utils.ExtraSugar(authGitHubStart)).Methods("GET").Name("authGitHubStart")
	r.HandleFunc("/auth/github/finish", utils.ExtraSugar(authGitHubFinish)).Methods("GET").Name("authGitHubFinish")

	r.HandleFunc("/auth/logout", utils.ExtraSugar(authLogout)).Methods("GET").Name("authLogout")
}

func authLoginStart(w http.ResponseWriter, r *http.Request) {
	oauthStart(w, r, "google", "/pastebin/auth/google", "openid", "profile")
}

func authGitHubStart(w http.ResponseWriter, r *http.Request) {
	oauthStart(w, r, "github", "/pastebin/auth/github/finish")
}

func authGitHubFinish(w http.ResponseWriter, r *http.Request) {
	_, err := oauthFinish(w, r, "github")
	if err != nil {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to talk to GitHub but something went wrong.", http.StatusInternalServerError)
		return
	}

	if token, err := models.GetOAuthToken(r); err == nil {
		client := &http.Client{}
		var responseData bytes.Buffer
		// https://developer.github.com/apps/building-oauth-apps/authorizing-oauth-apps/#3-use-the-access-token-to-access-the-api
		if request, err := http.NewRequest("GET", "https://api.github.com/user", &responseData); err == nil {
			request.Header.Set("Authorization", "token "+token.AccessToken)
			if response, err := client.Do(request); err == nil {
				defer response.Body.Close()
				if response.StatusCode == 200 {
					var user struct {
						ID int `json:"id"`
					}
					if data, err := ioutil.ReadAll(response.Body); err == nil {
						if err := json.Unmarshal([]byte(data), &user); err == nil {
							if err = utils.InitAppSession(w, r, strconv.Itoa(user.ID)); err == nil {
								utils.ClearOauthCookie(w)
								http.Redirect(w, r, "/pastebin/", http.StatusFound)
							} else {
								log.Printf("ERROR: %v\n", err)
								http.Error(w, "Meep! We were trying to initialize your session but something went wrong.", http.StatusInternalServerError)
							}
						} else {
							log.Printf("ERROR: %v\n", err)
							http.Error(w, "Meep! We were trying to parse your details but something went wrong.", http.StatusInternalServerError)
						}
					} else {
						log.Printf("ERROR: %v\n", err)
						http.Error(w, "Meep! We were trying to read your details but something went wrong.", http.StatusInternalServerError)
					}
				} else {
					if data, err := ioutil.ReadAll(response.Body); err != nil {
						log.Printf("ERROR: github returned non-OK, data could not be read, %v\n", err)
					} else {
						log.Printf("ERROR: github returned non-OK, %s\n", string(data))
					}
					http.Error(w, "Meep! We were trying to fetch your details but something isn't right.", response.StatusCode)
				}
			} else {
				log.Printf("ERROR: %v\n", err)
				http.Error(w, "Meep! We were trying to fetch your details but something went wrong.", http.StatusInternalServerError)
			}
		} else {
			log.Printf("ERROR: %v\n", err)
			http.Error(w, "Meep! We were trying to build a client for your token but something went wrong.", http.StatusInternalServerError)
		}
	} else {
		log.Printf("ERROR: %v\n", err)
		http.Error(w, "Meep! We were trying to fetch your token but something went wrong.", http.StatusInternalServerError)
	}
}

func authGoogle(w http.ResponseWriter, r *http.Request) {
	stateNonce, err := oauthFinish(w, r, "google", "openid", "profile")
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
				Nonce  string `json:"nonce"`
			}

			if strData, ok := data.(string); ok { // wow, golang can be awkward!
				// https://github.com/googleapis/google-auth-library-python/blob/08272d89667a901d0dff6c4ba53d5b30fcc29e81/google/auth/jwt.py#L134
				if strings.Count(strData, ".") != 2 {
					http.Error(w, "Meep! We were trying to count the dots in your token but something went wrong.", http.StatusInternalServerError)
					return
				}
				encodedData := strings.Split(strData, ".")[1]

				if jsonData, err := base64.RawURLEncoding.DecodeString(encodedData); err == nil {
					if err := json.Unmarshal([]byte(jsonData), &idToken); err == nil {
						if idToken.Nonce != stateNonce {
							log.Printf("WARNING: Nonce mismatch %s != %s", idToken.Nonce, stateNonce)
							http.Error(w, "Meep! We were trying to validate your session but something went wrong (nonce mismatch).", http.StatusBadRequest)
						} else if err = utils.InitAppSession(w, r, idToken.UserID); err == nil {
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
