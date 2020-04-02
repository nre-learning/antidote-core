#!/usr/bin/env bash

set -e

ASSETS_DIR=$1
UPLOAD_TARGET=$2

if [ -z "$ASSETS_DIR" ]
then
      echo "Must provide ASSETS_DIR"
      exit 1
fi

command -v jq > /dev/null
if [ $? -ne 0 ]; then
    echo "jq not installed"
    exit 1
fi

upload_github() {
    if [ -z "$GITHUB_TOKEN" ]
    then
          echo "Must provide GITHUB_TOKEN when specifying github as UPLOAD_TARGET"
          exit 1
    fi

    FILE=$1

    # Determine UPLOAD_URL
    UPLOAD_URL=$(curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/repos/nre-learning/antidote-core/releases/tags/$RELEASE_VERSION | jq .upload_url | sed -n 's/"\(.*\){.*/\1/p')

    # Upload asset
    echo "UPLOADING - $FILE"
    curl -H "Authorization: token $GITHUB_TOKEN" -H "Content-Type: $(file -b --mime-type $FILE)" --data-binary @$FILE \
    "$UPLOAD_URL?name=$(basename $FILE)"
}

upload_gcp() {
    FILE=$1
    echo "UPLOADING - $FILE"
    gsutil cp "$FILE" "gs://antidote-nightly-binaries/$(basename $FILE)"
}

shopt -s nullglob
for f in $ASSETS_DIR/*
do
  if [ ! -d $f ]
  then
    upload_$UPLOAD_TARGET $f
  fi
done
