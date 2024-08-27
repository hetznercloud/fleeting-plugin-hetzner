#!/usr/bin/env sh

set -e

if [ -z "${GPG_KEY_FILE}" ]; then
  echo "No GPG key file specified. Signing skipped."
  exit 0
fi

if [ -z "${GPG_KEY_PASSWORD_FILE}" ]; then
  echo "No GPG key password file specified. Signing skipped."
  exit 0
fi

gpg --batch --passphrase-file "${GPG_KEY_PASSWORD_FILE}" --import "${GPG_KEY_FILE}"
gpg  --pinentry-mode loopback --passphrase-file "${GPG_KEY_PASSWORD_FILE}" --armor --detach --sign "${CHECKSUMS_FILE}"
