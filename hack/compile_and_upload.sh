#!/usr/bin/env bash

set -e

GITHUB_TOKEN=$1
RELEASE_VERSION=$2

command -v jq > /dev/null
if [ $? -ne 0 ]; then
    echo "jq not installed"
    exit 1
fi

upload() {
    FILE=$1

    # Determine UPLOAD_URL
    UPLOAD_URL=$(curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/repos/nre-learning/antidote-core/releases/tags/$RELEASE_VERSION | jq .upload_url | sed -n 's/"\(.*\){.*/\1/p')

    # Upload asset
    echo "UPLOADING - $FILE"
    curl -H "Authorization: token $GITHUB_TOKEN" -H "Content-Type: $(file -b --mime-type $FILE)" --data-binary @$FILE \
    "$UPLOAD_URL?name=$(basename $FILE)"
}

mkdir -p $GOPATH/src/github.com/nre-learning
git clone --branch $RELEASE_VERSION https://github.com/nre-learning/antidote-core $GOPATH/src/github.com/nre-learning/antidote-core

# Make sure things compile and pass tests
cd $GOPATH/src/github.com/nre-learning/antidote-core
make

if [ $? -ne 0 ]; then
    echo "failed to compile"
    exit 1
fi

make test

if [ $? -ne 0 ]; then
    echo "failed to pass tests"
    exit 1
fi

cd -

packages=(
    "github.com/nre-learning/antidote-core/cmd/antidoted"
    "github.com/nre-learning/antidote-core/cmd/antictl"
    "github.com/nre-learning/antidote-core/cmd/antidote"
)

# https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04
platforms=("windows/amd64" "windows/386" "darwin/amd64" "linux/amd64")

for platform in "${platforms[@]}"
do

    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    archive_name='antidote-'$GOOS'-'$GOARCH

    for package in "${packages[@]}"
    do

        echo "Compiling $package..."
        package_split=(${package//\// })
        package_name=${package_split[-1]}

        directory_name='release_assets/antidote-'$GOOS'-'$GOARCH
        output_name=$package_name
        if [ $GOOS = "windows" ]; then
            output_name+='.exe'
        fi  

        echo "Compiling for platform $platform, at $directory_name/$output_name"
        env GOOS=$GOOS GOARCH=$GOARCH go build -o $directory_name/$output_name $package
        if [ $? -ne 0 ]; then
            echo 'An error has occurred! Aborting the script execution...'
            exit 1
        fi

        echo "Adding to ZIP archive..."
        zip -j 'release_assets/'$archive_name'.zip' $directory_name/$output_name

    done

    tar -czvf 'release_assets/'$archive_name'.tar.gz'  -C $directory_name .
    # extract with tar xvzf file.tar.gz

    echo "$(sha256sum 'release_assets/'$archive_name'.zip' | awk '{print $1}') $archive_name.zip" >> release_assets/hashes.txt
    echo "$(sha256sum 'release_assets/'$archive_name'.tar.gz' | awk '{print $1}') $archive_name.tar.gz" >> release_assets/hashes.txt

    upload 'release_assets/'$archive_name'.zip'
    upload 'release_assets/'$archive_name'.tar.gz'

done

cat release_assets/hashes.txt
