package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pgaskin/kobopatch/patchlib"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "symdump dumps symbol addresses from an ARMv6+ 32-bit ELF executable")
		fmt.Fprintln(os.Stderr, "Usage: symdump BINARY_FILE")
		os.Exit(1)
	}

	buf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	pt := patchlib.NewPatcher(buf)

	ds, err := pt.ExtractDynsyms(true)
	if err != nil {
		panic(err)
	}

	f, err := os.Create("symdump.out.json")
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(f, "[\n")
	for i, s := range ds {
		if i != 0 {
			fmt.Fprintf(f, ",\n")
		}
		buf, _ := json.Marshal(s)
		f.Write(buf)
	}
	fmt.Fprintf(f, "]\n")

	f.Close()
	os.Exit(0)
}
