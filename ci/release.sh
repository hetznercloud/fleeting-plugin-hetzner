#!/bin/bash

# This script assumes it is running within a
# registry.gitlab.com/gitlab-org/release-cli:latest image, or at least that
# `release-cli` is installed and in $PATH. Also note that this is very much a
# bash script and does not run under plain sh.

set -e

args=( create --name "Release $CI_COMMIT_TAG" --tag-name "$CI_COMMIT_TAG" )
while read -r BIN
do
    # Note: change "filepath" to "direct_asset_path" when https://gitlab.com/gitlab-org/release-cli/-/issues/165 is fixed.
    args+=( --assets-link "{\"name\":\"${BIN}\",\"url\":\"${PACKAGE_REGISTRY_URL}/${CI_COMMIT_TAG}/${BIN}\", \"filepath\":\"/${BIN}\"}" )
done < manifest.txt

release-cli "${args[@]}"
