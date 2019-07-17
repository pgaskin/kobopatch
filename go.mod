module github.com/geek1011/kobopatch

go 1.12

require (
	github.com/DataDog/czlib v0.0.0-20160811164712-4bc9a24e37f2
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ianlancetaylor/demangle v0.0.0-20181102032728-5e5cf60278f6
	github.com/pkg/errors v0.8.1
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v0.0.0-20181212140101-916c5bf2d89a
	github.com/stretchr/testify v0.0.0-20180319223459-c679ae2cc0cb
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8
	gopkg.in/yaml.v2 v2.2.2
	gopkg.in/yaml.v3 v3.0.0-20190709130402-674ba3eaed22
)

replace gopkg.in/yaml.v3 => github.com/geek1011/yaml v0.0.0-20190717135119-db0123c0912e // v3-node-decodestrict
