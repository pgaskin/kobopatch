module github.com/geek1011/kobopatch

go 1.13

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/geek1011/czlib v0.0.3
	github.com/ianlancetaylor/demangle v0.0.0-20181102032728-5e5cf60278f6
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8
	gopkg.in/yaml.v3 v3.0.0-20190709130402-674ba3eaed22
	rsc.io/arm v0.0.0-20150420010332-9c32f2193064
)

replace gopkg.in/yaml.v3 => github.com/geek1011/yaml v0.0.0-20190717135119-db0123c0912e // v3-node-decodestrict
