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
	go build -v -x -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract" github.com/geek1011/kobopatch/cssextract
	go build -v -x -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/patch32lsb" github.com/geek1011/kobopatch/patch32lsb
	go build -v -x -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/p32lsb2kobopatch" github.com/geek1011/kobopatch/p32lsb2kobopatch

.PHONY: cross
cross:
	mkdir -p build
	# Needs osxcross, gcc-7-multilib-i686-linux-gnu, gcc-mingw-w64-i686, zlib1g-dev:i386, libz-mingw-w64-dev, gcc-7-arm-linux-gnueabihf, clang
	CC=i686-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -ldflags "-linkmode external -extldflags -static -X main.version=$(shell git describe --tags --always)" -o "build/koboptch-windows.exe" github.com/geek1011/kobopatch/kobopatch
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-linux-64bit" github.com/geek1011/kobopatch/kobopatch
	CC=i686-linux-gnu-gcc-7 CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-linux-32bit" github.com/geek1011/kobopatch/kobopatch
	# CC=arm-linux-gnueabihf-gcc-7 CGO_ENABLED=1 GOOS=linux GOARCH=arm go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-linux-arm" github.com/geek1011/kobopatch/kobopatch
	CC=o64-clang CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-darwin-64bit" github.com/geek1011/kobopatch/kobopatch

	CC=i686-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -ldflags "-linkmode external -extldflags -static -X main.version=$(shell git describe --tags --always)" -o "build/cssextract-windows.exe" github.com/geek1011/kobopatch/cssextract
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract-linux-64bit" github.com/geek1011/kobopatch/cssextract
	CC=i686-linux-gnu-gcc-7 CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract-linux-32bit" github.com/geek1011/kobopatch/cssextract
	# CC=arm-linux-gnueabihf-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract-linux-arm" github.com/geek1011/kobopatch/cssextract
	CC=o64-clang CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract-darwin-64bit" github.com/geek1011/kobopatch/cssextract

.PHONY: convert
convert:
	./converter/convert.sh