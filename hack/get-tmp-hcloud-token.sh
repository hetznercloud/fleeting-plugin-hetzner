#!/usr/bin/env bash

set -eu

error() {
  echo >&2 "error: $*"
  exit 1
}

command -v curl > /dev/null || error "curl command not found!"

curl \
  --silent \
  --fail-with-body \
  --request "POST" \
  --user-agent "tps-client/unknown" \
  --header "Authorization: Bearer $TPS_TOKEN" \
  "$TPS_URL" || error "could not generate temporary token!"
