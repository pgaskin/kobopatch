// Command kobopatch-apply applies a single patch file to a binary.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/geek1011/kobopatch/patchlib"

	"github.com/geek1011/kobopatch/patchfile"
	_ "github.com/geek1011/kobopatch/patchfile/kobopatch"
	_ "github.com/geek1011/kobopatch/patchfile/patch32lsb"
	"github.com/spf13/pflag"
)

var version = "unknown"

func errexit(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func main() {
	input := pflag.StringP("input", "i", "", "the file to patch (required)")
	patchFile := pflag.StringP("patch-file", "p", "", "the file containing the patches (required)")
	output := pflag.StringP("output", "o", "", "the file to write the patched output to (will be overwritten if exists) (required)")
	patchFormat := pflag.StringP("patch-format", "f", "kobopatch", fmt.Sprintf("the patch format (one of: %s)", strings.Join(patchfile.GetFormats(), ",")))
	verbose := pflag.BoolP("verbose", "v", false, "show verbose output from patchlib")
	help := pflag.BoolP("help", "h", false, "show this help text")
	pflag.Parse()

	if *help || pflag.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "Usage: kobopatch-apply [OPTIONS]\n")
		fmt.Fprintf(os.Stderr, "\nVersion: %s\n\nOptions:\n", version)
		pflag.PrintDefaults()
		os.Exit(1)
	}

	if *input == "" || *patchFile == "" || *output == "" {
		errexit("Error: input, patch-file, and output flags are required. See --help for more info.\n")
	}

	if !sliceContains(patchfile.GetFormats(), *patchFormat) {
		errexit("Error: invalid format %s. See --help for more info.\n", *patchFormat)
	}

	if *verbose {
		patchfile.Log = func(format string, a ...interface{}) {
			fmt.Printf(format, a...)
		}
	} else {
		patchfile.Log = func(format string, a ...interface{}) {}
	}

	ps, err := patchfile.ReadFromFile(*patchFormat, *patchFile)
	if err != nil {
		errexit("Error: could not read patch file: %v\n", err)
	}

	err = ps.Validate()
	if err != nil {
		errexit("Error: could not validate patch file: %v\n", err)
	}

	buf, err := ioutil.ReadFile(*input)
	if err != nil {
		errexit("Error: could not read input file: %v\n", err)
	}

	pt := patchlib.NewPatcher(buf)

	err = ps.ApplyTo(pt)
	if err != nil {
		errexit("Error: could not apply patch file: %v\n", err)
	}

	f, err := os.Create(*output)
	if err != nil {
		errexit("Error: could not create output file: %v\n", err)
	}

	obuf := pt.GetBytes()
	n, err := f.Write(obuf)
	if err != nil {
		errexit("Error: could not write output file: %v\n", err)
	} else if n != len(obuf) {
		errexit("Error: could not create output file: could not finish writing all bytes to file\n")
	}

	fmt.Printf("Successfully patched '%s' using '%s' to '%s'\n", *input, *patchFile, *output)
	os.Exit(0)
}

func sliceContains(arr []string, v string) bool {
	for _, i := range arr {
		if i == v {
			return true
		}
	}
	return false
}
