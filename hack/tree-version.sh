#! /usr/bin/env bash

set -o pipefail
set -o nounset
set -o errexit

git describe --exact-match 2>/dev/null || \
    basename $( git describe --all --long 2>/dev/null)
