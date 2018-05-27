.PHONY: default
default: clean build-deps deps generate test build

.PHONY: clean
clean:
	rm -rf build

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