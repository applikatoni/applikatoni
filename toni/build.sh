#!/bin/bash
# Builds and packages the toni binary

set -e

target="toni-$(go env GOOS)-$(go env GOARCH)"
executable="toni"
current_revision=$(git rev-parse HEAD)

mkdir ./builds/$target

go build -o ./builds/$target/$executable ./ || exit 1

cp ./README.md ./builds/$target/
cp ../LICENSE ./builds/$target/
echo ${current_revision} >> ./builds/$target/VERSION

tar czvfC ./builds/$target.tar.gz ./builds $target

rm -rf ./builds/$target
