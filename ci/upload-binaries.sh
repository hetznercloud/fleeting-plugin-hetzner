#!/bin/sh

set -e

OUT_PATH="${OUT_PATH:-bin}"

for BIN in "${OUT_PATH}"/*
do
    URL="$PACKAGE_REGISTRY_URL/${CI_COMMIT_TAG}/$(basename "$BIN")"
    echo "Uploading ${BIN} to ${URL}"
    curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file "$BIN" "$URL"
done

# List the filenames uploaded so we can use them in the release job
ls "$OUT_PATH" > manifest.txt
