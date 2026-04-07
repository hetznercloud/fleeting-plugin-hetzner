#!/usr/bin/env bash

# This script assumes it is running within a registry.gitlab.com/gitlab-org/cli:latest image

set -euo pipefail

# asset_link <file>
asset_link() {
  cat << EOF
{"name": "${1}", "url": "${PACKAGE_REGISTRY_URL}/${CI_COMMIT_TAG}/${1}", "direct_asset_path": "/${1}"}
EOF
}

# assets_links
assets_links() {
  assets=()
  while read -r file; do
    assets+=("$(asset_link "$file")")
  done < manifest.txt
  local IFS=,
  echo "[${assets[*]}]"
}

glab release upload "${CI_COMMIT_TAG}" --assets-links "$(assets_links)"
