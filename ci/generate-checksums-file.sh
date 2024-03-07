#!/usr/bin/env sh

set -e

cd "${OUT_PATH}"
sha256sum "${NAME}"-* > "${CHECKSUMS_FILE_NAME}"
