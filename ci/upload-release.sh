#!/usr/bin/env sh

set -e

OUT_PATH="${OUT_PATH:-out}"

for FILE in "${OUT_PATH}"/*; do
  URL="${PACKAGE_REGISTRY_URL}/${CI_COMMIT_TAG}/$(basename "${FILE}")"
  echo "Uploading ${FILE} to ${URL}"
  curl --header "JOB-TOKEN: ${CI_JOB_TOKEN}" --upload-file "${FILE}" "${URL}"
done

# List the filenames uploaded so we can use them in the release job
ls "${OUT_PATH}" > manifest.txt
