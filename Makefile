.PHONY: default
default: clean build-deps deps generate test build

.PHONY: clean
clean:
	rm -rf build
	rm -rf converter/out

.PHONY: build-deps
build-deps:
	go get -v "github.com/kardianos/govendor"
	go get -v "github.com/aktau/github-release"

.PHONY: deps
deps:
	govendor sync

.PHONY: generate
generate:
	go generate ./...

.PHONY: test
test:
	go test -v ./...

.PHONY: build
build:
	mkdir -p build
	go build -v -x -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch" github.com/geek1011/kobopatch/kobopatch

.PHONY: cross
cross:
	GOOS=windows GOARCH=386 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-windows.exe" github.com/geek1011/kobopatch/kobopatch
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-linux-64bit" github.com/geek1011/kobopatch/kobopatch
	GOOS=linux GOARCH=386 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-linux-32bit" github.com/geek1011/kobopatch/kobopatch
	GOOS=linux GOARCH=arm go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-linux-arm" github.com/geek1011/kobopatch/kobopatch
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-darwin-64bit" github.com/geek1011/kobopatch/kobopatch

.PHONY: convert
convert:
	./converter/convert.sh