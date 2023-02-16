#!/usr/bin/env sh

set -e

checksumsFile="${OUT_PATH}/release.sha256"
rm -f "${checksumsFile}"

for FILE in "${OUT_PATH}"/*
do
  sha256sum "${FILE}" >> "${checksumsFile}"
done
