#!/usr/bin/env bash

set -eo pipefail

# convert 'out/fleeting-plugin-{name}-{os}-{arch}' to 'dist/{os}/{arch}/plugin'
for path in ./out/fleeting-plugin-*; do
  basename=$(basename "$path")
  IFS='-' read -ra parts <<< "${basename}"

  os=${parts[3]}
  arch=${parts[4]%.exe}

  if [ "$arch" = "arm" ]; then
    arch="armv7"
  fi

  ext=""
  if [ "$os" = "windows" ]; then
    ext=".exe"
  fi

  mkdir -p "dist/${os}/${arch}/"
  mv "${path}" "dist/${os}/${arch}/plugin${ext}"
done

find ./dist

go install gitlab.com/gitlab-org/fleeting/fleeting-artifact/cmd/fleeting-artifact@latest

VERSION=${CI_COMMIT_TAG:=0.0.0-bleeding}

# login to registry
fleeting-artifact login -username "$CI_REGISTRY_USER" -password "$CI_REGISTRY_PASSWORD" "$CI_REGISTRY"

# releast artifact
fleeting-artifact release "$CI_REGISTRY_IMAGE:${VERSION#v}"
