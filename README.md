[![Build Status](https://travis-ci.org/adayoung/gae-pastebin.svg?branch=noGoogleBranch)](https://travis-ci.org/adayoung/gae-pastebin)

# gae-pastebin
Hi! Here's a pastebin that runs ~on Google App Engine~ anywhere!

## Prerequisites

 * A working [Go](https://golang.org/doc/install) environment, preferably >go1.11
 * An account with a PostgreSQL server with credentials noted in keys.yaml
 * An account with the [Google reCAPTCHA](https://www.google.com/recaptcha/) project with site key and secret key noted in keys.yaml
 * An account with [Google Cloud Platform](https://cloud.google.com/) with [Google Drive API (v3)](https://developers.google.com/drive/) enabled, credentials in keys.yaml
 * An account with [Cloudflare](https://www.cloudflare.com/) with an API Token scopred for `Zone.Cache Purge`, credentials in keys.yaml

## How to use

 * Get the package and its dependencies with `go get github.com/adayoung/gae-pastebin`
 * `cd $GOPATH/src/github.com/adayoung/gae-pastebin`, `go run .`
 * Point your brower to http://localhost:2019/
 * Sample deployment stuffs are available in confs/

_Oh, make sure to update static/js/base.js with your own GA user id!_  
_And keys.yaml to change the CSRFAuthKey and EncryptionK as well!_
