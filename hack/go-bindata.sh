#! /usr/bin/env bash

# Wrap go-bindata. We used to just to "go run", but that now fails
# to resolve all the packages. More modules suckiness I guess.

set -o pipefail
set -o nounset
set -o errexit

export GO111MODULE=on

if ! command -v go-bindata >/dev/null ; then
	go get -u github.com/go-bindata/go-bindata/...
fi

go-bindata "$@"
