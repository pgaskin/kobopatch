#!/bin/bash

cd "$(dirname "$0")"
rm -rf build
mkdir build
cd build
GOOS=windows GOARCH=amd64 go build -o "kobopatch-windows-64bit.exe" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go
GOOS=windows GOARCH=386 go build -o "kobopatch-windows-32bit.exe" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go
GOOS=linux GOARCH=amd64 go build -o "kobopatch-linux-64bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go
GOOS=linux GOARCH=386 go build -o "kobopatch-linux-32bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go
GOOS=linux GOARCH=arm go build -o "kobopatch-linux-arm" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go
GOOS=darwin GOARCH=amd64 go build -o "kobopatch-darwin-64bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go
GOOS=darwin GOARCH=386 go build -o "kobopatch-darwin-32bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go
GOOS=freebsd GOARCH=amd64 go build -o "kobopatch-freebsd-64bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go
GOOS=freebsd GOARCH=386 go build -o "kobopatch-freebsd-32bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go
GOOS=freebsd GOARCH=arm go build -o "kobopatch-freebsd-arm" -ldflags "-s -w -X main.version=$(git describe --tags --always)" kobopatch.go