package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/geek1011/kobopatch/patchlib"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "cssextract extracts zlib-compressed from a binary file")
		fmt.Fprintln(os.Stderr, "Usage: cssextract BINARY_FILE")
	}

	buf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	pt := patchlib.NewPatcher(buf)

	z, err := pt.ExtractZlib()
	if err != nil {
		panic(err)
	}

	f, err := os.Create("cssextract.out.css")
	if err != nil {
		panic(err)
	}

	for offset, css := range z {
		fmt.Fprintf(f, "/* zlib stream: offset_hex(0x%X) offset_int32(%d) len_int32(%d) sha1(%x) */\n%s\n\n", offset, offset, len(css), sha1.Sum([]byte(css)), css)
	}

	f.Close()
	os.Exit(0)
}
