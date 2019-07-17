.PHONY: default
default: clean build-deps deps generate test build

.PHONY: clean
clean:
	rm -rf build
	rm -rf converter/out

.PHONY: build-deps
build-deps:
	go get -v "github.com/aktau/github-release"

.PHONY: deps
deps:
	go mod download

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
	go build -v -x -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-apply" github.com/geek1011/kobopatch/tools/kobopatch-apply
	go build -v -x -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-mkzlib" github.com/geek1011/kobopatch/tools/kobopatch-mkzlib
	go build -v -x -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract" github.com/geek1011/kobopatch/tools/cssextract

.PHONY: cross
cross:
	mkdir -p build
	# debian: osxcross, gcc-7-multilib-i686-linux-gnu, gcc-mingw-w64-i686, zlib1g-dev:i386, libz-mingw-w64-dev, gcc-7-arm-linux-gnueabihf, clang
	# ubuntu trusty: osxcross, gcc-multilib, gcc-mingw-w64-i686, zlib1g-dev:i386, libz-mingw-w64-dev, gcc-7-arm-linux-gnueabihf, clang, libc6-dev-i386
	CC=i686-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -ldflags "-linkmode external -extldflags -static -X main.version=$(shell git describe --tags --always)" -o "build/koboptch-windows.exe" github.com/geek1011/kobopatch/kobopatch
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-linux-64bit" github.com/geek1011/kobopatch/kobopatch
	CC="gcc -m32" CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-linux-32bit" github.com/geek1011/kobopatch/kobopatch
	# CC=arm-linux-gnueabihf-gcc-7 CGO_ENABLED=1 GOOS=linux GOARCH=arm go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-linux-arm" github.com/geek1011/kobopatch/kobopatch
	CC=o64-clang CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-darwin-64bit" github.com/geek1011/kobopatch/kobopatch

	CC=i686-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -ldflags "-linkmode external -extldflags -static -X main.version=$(shell git describe --tags --always)" -o "build/koboptch-apply-windows.exe" github.com/geek1011/kobopatch/tools/kobopatch-apply
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-apply-linux-64bit" github.com/geek1011/kobopatch/tools/kobopatch-apply
	CC="gcc -m32" CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-apply-linux-32bit" github.com/geek1011/kobopatch/tools/kobopatch-apply
	# CC=arm-linux-gnueabihf-gcc-7 CGO_ENABLED=1 GOOS=linux GOARCH=arm go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-apply-linux-arm" github.com/geek1011/kobopatch/tools/kobopatch-apply
	CC=o64-clang CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/kobopatch-apply-darwin-64bit" github.com/geek1011/kobopatch/tools/kobopatch-apply

	CC=i686-w64-mingw32-gcc CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -ldflags "-linkmode external -extldflags -static -X main.version=$(shell git describe --tags --always)" -o "build/cssextract-windows.exe" github.com/geek1011/kobopatch/tools/cssextract
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract-linux-64bit" github.com/geek1011/kobopatch/tools/cssextract
	CC="gcc -m32" CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract-linux-32bit" github.com/geek1011/kobopatch/tools/cssextract
	# CC=arm-linux-gnueabihf-gcc CGO_ENABLED=1 GOOS=linux GOARCH=arm go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract-linux-arm" github.com/geek1011/kobopatch/tools/cssextract
	CC=o64-clang CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(shell git describe --tags --always)" -o "build/cssextract-darwin-64bit" github.com/geek1011/kobopatch/tools/cssextract
