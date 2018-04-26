#!/bin/bash

cd "$(dirname "$0")"
rm -rf build
mkdir build
cd build
GOOS=windows GOARCH=amd64 go build -o "patch32lsb-windows-64bit.exe" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go
GOOS=windows GOARCH=386 go build -o "patch32lsb-windows-32bit.exe" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go
GOOS=linux GOARCH=amd64 go build -o "patch32lsb-linux-64bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go
GOOS=linux GOARCH=386 go build -o "patch32lsb-linux-32bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go
GOOS=linux GOARCH=arm go build -o "patch32lsb-linux-arm" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go
GOOS=darwin GOARCH=amd64 go build -o "patch32lsb-darwin-64bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go
GOOS=darwin GOARCH=386 go build -o "patch32lsb-darwin-32bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go
GOOS=freebsd GOARCH=amd64 go build -o "patch32lsb-freebsd-64bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go
GOOS=freebsd GOARCH=386 go build -o "patch32lsb-freebsd-32bit" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go
GOOS=freebsd GOARCH=arm go build -o "patch32lsb-freebsd-arm" -ldflags "-s -w -X main.version=$(git describe --tags --always)" ../patch32lsb.go