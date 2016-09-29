# gae-pastebin
Hi! Here's a pastebin that runs on Google App Engine!

## Prerequisites

 * [Google App Engine Go SDK](https://cloud.google.com/appengine/downloads)
 * [github.com/gorilla/mux](http://www.gorillatoolkit.org/pkg/mux)
 * [github.com/gorilla/csrf](http://www.gorillatoolkit.org/pkg/csrf)

## How to use

 * You'll need to download and extract [Google App Engine Go SDK](https://cloud.google.com/appengine/downloads) for Go
 * Clone the repository and initialize submodules with:
   * `git submodule init`
   * `git submodule update`
 * Go get dependencies with:
   * `env GOPATH=<sdk path>/gopath go get github.com/gorilla/mux`
   * `env GOPATH=<sdk path>/gopath go get github.com/gorilla/csrf`
 * Now launch the App Engine Development Server and you're good to go!
   * `<sdk path>/goapp serve gae-pastebin`
 * Deploy to your own Google account with:
   * `<sdk path>/goapp deploy -application [YOUR_PROJECT_ID] -version [YOUR_VERSION_ID]`

_Oh, make sure to update static/js/base.js with your own GA user id!_
_And pastebin/pastebin.go to change the csrf_auth_key as well!_

## TODO

 * Loading app secrets/config from external sources
 * Implement the /search route again
 * Enable HTTPS again
