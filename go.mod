module github.com/pgaskin/kobopatch

go 1.23.12

require (
	github.com/ianlancetaylor/demangle v0.0.0-20250628045327-2d64ad6b7ec5
	github.com/pgaskin/go-libz v0.0.2
	github.com/riking/cssparse v0.0.0-20180325025645-c37ded0aac89
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.11.1
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8
	gopkg.in/yaml.v3 v3.0.1
	rsc.io/arm v0.0.0-20150420010332-9c32f2193064
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.9.0 // indirect
	golang.org/x/text v0.3.2 // indirect
)

replace gopkg.in/yaml.v3 => github.com/pgaskin/yaml v0.0.0-20190717135119-db0123c0912e // v3-node-decodestrict
