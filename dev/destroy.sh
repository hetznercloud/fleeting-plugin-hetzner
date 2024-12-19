#!/usr/bin/env bash

set -eu -o pipefail

error() {
  echo >&2 "error: $*"
  exit 1
}

command -v hcloud > /dev/null || error "hcloud command not found!"

servers=$(hcloud server list -l instance-group=dev-docker-autoscaler -o json | jq '.[].id')
volumes=$(hcloud volume list -l instance-group=dev-docker-autoscaler -o json | jq '.[].id')

if [[ -n "$servers" ]]; then
  # shellcheck disable=SC2086
  hcloud server delete $servers
fi

if [[ -n "$volumes" ]]; then
  # shellcheck disable=SC2086
  hcloud volume delete $volumes
fi
