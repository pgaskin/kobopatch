package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnescape(t *testing.T) {
	for _, c := range [][]string{
		{`test`, "test"},
		{`test\0`, "test\x00"},
		{`test\0\xcc`, "test\x00\xcc"},
		{`test\0\n\t\v\r\xcc`, "test\x00\n\t\v\r\xcc"},
		{`test\0\n\t\v\r\xcc\"\'\` + "`", "test\x00\n\t\v\r\xcc\"'`"},
	} {
		u, err := unescape("`" + c[0] + "`")
		assert.NoError(t, err)
		assert.Equal(t, c[1], u)
	}
}
