#!/bin/bash

set -e

echo -e "\n-- $(date) -- Starting --\n"

[ -d build ] && rm -rf build
mkdir -p build/libs

python ./build.py
tar xzf ./libs/go-sqlite3-*.tar.gz -C ./build/libs/
mv ./build/libs/go-sqlite3-1.10.0 ./build/libs/go-sqlite3

echo -e "\n-- $(date) -- Building --\n"

ldflags="-linkmode=external -extldflags=-static"
go build -ldflags="${ldflags}" -o ./build/lnxmonsrv ./build/lnxmonsrv.go
go build -ldflags="${ldflags}" -o ./build/lnxmoncli ./build/lnxmoncli.go

[ -f ./build/lnxmonsrv.go ] && rm ./build/lnxmonsrv.go
[ -f ./build/lnxmoncli.go ] && rm ./build/lnxmoncli.go

[ -d build/libs ] && rm -rf build/libs

echo -e "\n-- $(date) -- Finished --\n"
