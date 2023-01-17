#!/bin/bash

# This script assumes it is running within a
# registry.gitlab.com/gitlab-org/release-cli:latest image, or at least that
# `release-cli` is installed and in $PATH. Also note that this is very much a
# bash script and does not run under plain sh.

set -e

args=( create --name "Release $CI_COMMIT_TAG" --tag-name "$CI_COMMIT_TAG" )
while read -r BIN
do
    args+=( --assets-link "{\"name\":\"${BIN}\",\"url\":\"${PACKAGE_REGISTRY_URL}/${CI_COMMIT_TAG}/${BIN}\"}" )
done < manifest.txt


# It's not possible to update an existing release to point to new artifacts, so
# we have to delete the existing latest release before creating a new one.
if [[ "${CI_COMMIT_TAG}" == "latest" ]]
then
  curl --request DELETE --header "JOB-TOKEN: $CI_JOB_TOKEN" "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/releases/latest"
fi

release-cli "${args[@]}"
