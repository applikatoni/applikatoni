#!/bin/bash
# Builds and packages the applikatoni server binary

set -e

version=$(cat main.go | grep "VERSION\s*=" | awk '{print $NF}' | sed 's/\"//g')
target="applikatoni-${version}-$(go env GOOS)-$(go env GOARCH)"
executable="applikatoni"
goose_executable=$(which goose)
current_revision=$(git rev-parse HEAD)

rm -rf ./builds/$target
mkdir ./builds/$target

go build -o ./builds/$target/$executable ./ || exit 1

mkdir -p ./builds/$target/db/

cp ./db/dbconf.yml ./builds/$target/db/
cp -R ./db/migrations ./builds/$target/db/
cp ./configuration_example.json ./builds/$target/
cp -R ./assets ./builds/$target/
cp ../LICENSE ./builds/$target/
cp ../README.md ./builds/$target/

echo ${version} >> ./builds/$target/VERSION

cp ${goose_executable} ./builds/$target/

tar czvfC ./builds/$target.tar.gz ./builds $target

rm -rf ./builds/$target
