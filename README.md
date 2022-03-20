[![Go](https://github.com/adayoung/gae-pastebin/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/adayoung/gae-pastebin/actions/workflows/go.yml)
[![Go report](https://goreportcard.com/badge/adayoung/gae-pastebin)](https://goreportcard.com/report/adayoung/gae-pastebin)

# gae-pastebin
Hi! Here's a pastebin that runs ~on Google App Engine~ anywhere!

## Prerequisites

 * A working [Go](https://golang.org/doc/install) environment, preferably >go1.11
 * An account with a PostgreSQL server with credentials noted in keys.yaml
 * Access to a [Redis](https://redis.io/) instance without password
 * An account with the [Google reCAPTCHA](https://www.google.com/recaptcha/) project with site key and secret key noted in keys.yaml
 * An account with [Google Cloud Platform](https://cloud.google.com/) with [Google Drive API (v3)](https://developers.google.com/drive/) enabled, credentials in keys.yaml
 * An account with [Cloudflare](https://www.cloudflare.com/) with an API Token scoped for `Zone.Cache Purge`, credentials in keys.yaml

## How to use

 * Get the package and its dependencies with `go get github.com/adayoung/gae-pastebin`
 * `cd $GOPATH/src/github.com/adayoung/gae-pastebin`, `go run .`
 * Point your brower to http://localhost:2019/
 * Sample deployment stuffs are available in confs/

## Building with Docker

 Use the following command to build with the latest version of Go:
  * `cd <path to repository>`
  * `docker run --rm -v $PWD:/go/src/github.com/adayoung/gae-pastebin -w /go/src/github.com/adayoung/gae-pastebin -v $GOPATH:/go -e "CGO_ENABLED=0" golang:latest go build -v -ldflags "-s -w" .`

_And keys.yaml to change the CSRFAuthKey and EncryptionK as well!_

_P.S.: This [used to](https://github.com/adayoung/gae-pastebin/releases/tag/v2019-09-29) run on Google App Engine and there's probably a bunch of stuff about it that still lingers on. I'll eventually clean it up :open_mouth:_
