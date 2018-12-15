// Command kobopatch-mkzlib creates an old-style replace_bytes based patch32lsb patch for a zlib patch.
package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/geek1011/kobopatch/patchfile"
	"github.com/geek1011/kobopatch/patchfile/kobopatch"
	_ "github.com/geek1011/kobopatch/patchfile/kobopatch"
	_ "github.com/geek1011/kobopatch/patchfile/patch32lsb"
	"github.com/geek1011/kobopatch/patchlib"
	"github.com/spf13/pflag"
)

var version = "unknown"

func errexit(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func main() {
	input := pflag.StringP("input", "i", "", "the original binary (required)")
	patchFile := pflag.StringP("patch-file", "p", "", "the file containing the patches in kobopatch format (required)")
	patchName := pflag.StringP("patch-name", "n", "", "the name of the patch to convert (required)")
	help := pflag.BoolP("help", "h", false, "show this help text")
	pflag.Parse()

	if *help || pflag.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "Usage: kobopatch-mkzlib [OPTIONS]\n")
		fmt.Fprintf(os.Stderr, "\nVersion: %s\n\nOptions:\n", version)
		pflag.PrintDefaults()
		os.Exit(1)
	}

	if *input == "" || *patchFile == "" || *patchName == "" {
		errexit("Error: input, patch-file, and patch-name flags are required. See --help for more info.\n")
	}

	fmt.Printf("\nAPPLYING PATCH TO FIND CHANGES:\n")
	ps, err := patchfile.ReadFromFile("kobopatch", *patchFile)
	if err != nil {
		errexit("Error: could not read patch file: %v\n", err)
	}
	kps := ps.(*kobopatch.PatchSet)
	found := false
	kpatchGroup := ""
	kpatchDesc := ""
	for n, insts := range *kps {
		m := n == *patchName
		err := ps.SetEnabled(n, m)
		if err != nil {
			errexit("Error: could not SetEnabled patch '%s': unknown error: %v\n", n, err)
		}
		found = found || m
		if m {
			for _, inst := range insts {
				if inst.PatchGroup != nil {
					kpatchGroup = *inst.PatchGroup
				} else if inst.Description != nil {
					kpatchDesc = *inst.Description
				}
			}
		}
	}
	if !found {
		errexit("Error: could not enable patch '%s': no such patch\n", *patchName)
	}
	err = ps.Validate()
	if err != nil {
		errexit("Error: could not validate patch file: %v\n", err)
	}
	buf, err := ioutil.ReadFile(*input)
	if err != nil {
		errexit("Error: could not read input file: %v\n", err)
	}
	ibuf := make([]byte, len(buf))
	copy(ibuf, buf)
	pt := patchlib.NewPatcher(buf)
	err = ps.ApplyTo(pt)
	if err != nil {
		errexit("Error: could not apply patch file: %v\n", err)
	}
	obuf := pt.GetBytes()
	if len(ibuf) != len(obuf) {
		errexit("Error: unhandled case: len(ibuf) != len(obuf). Please report this as a bug.\n")
	}
	if bytes.Equal(ibuf, obuf) {
		errexit("Error: no changes made.\n")
	}
	fmt.Printf("--> SUCCESS\n")

	fmt.Printf("\nGENERATING NEW PATCH:\n")
	var patch string
	outf := func(format string, a ...interface{}) {
		patch += fmt.Sprintf(format+"\n", a...)
	}
	outf("<Patch>")
	outf("patch_name = `%s`", *patchName)
	outf("patch_enable = `no`")
	outf("#")
	outf("# patch_generator = `kobopatch-mkzlib`")
	if kpatchGroup != "" {
		outf("# patch_group = `%s`", kpatchGroup)
	}
	if kpatchDesc != "" {
		outf("#")
		for _, l := range strings.Split(strings.Trim(strings.Replace(kpatchDesc, "\r\n", "\n", -1), "\n"), "\n") {
			outf("## %s", l)
		}
	}
	outf("#")
	replacements := []replacement{}
	for i := range ibuf {
		if ibuf[i] != obuf[i] {
			if cur := len(replacements) - 1; cur > -1 && replacements[cur].Offset == i-len(replacements[cur].Find) && len(replacements[cur].Find) < 15 {
				// Right after previous replacement and previous replacement less than 15 bytes long.
				replacements[cur].Find = append(replacements[cur].Find, ibuf[i])
				replacements[cur].Replace = append(replacements[cur].Replace, obuf[i])
			} else {
				replacements = append(replacements, replacement{
					Offset:  i,
					Find:    []byte{ibuf[i]},
					Replace: []byte{obuf[i]},
				})
			}
		} else if cur := len(replacements) - 1; cur > -1 && i+1 < len(ibuf) && ibuf[i-1] != obuf[i-1] && ibuf[i] == obuf[i] && ibuf[i+1] != obuf[i+1] && len(replacements[cur].Find) < 15 {
			// Only a 1 byte difference separating replacements and previous replacement less than 15 bytes long.
			replacements[cur].Find = append(replacements[cur].Find, ibuf[i])
			replacements[cur].Replace = append(replacements[cur].Replace, obuf[i])
		}
	}
	outf("base_address = 0")
	for _, r := range replacements {
		outf("replace_bytes = %07X, % X, % X", r.Offset, r.Find, r.Replace)
	}
	outf("</Patch>")
	fmt.Printf("--> SUCCESS\n")

	fmt.Printf("\nTESTING GENERATED PATCH:\n")
	patch32lsb, ok := patchfile.GetFormat("patch32lsb")
	if !ok {
		errexit("Error: internal error: could not load patch32lsb format. Please report this as a bug.\n")
	}
	p32ps, err := patch32lsb([]byte(strings.Replace(patch, "# patch_group", "patch_group", 1)))
	if err != nil {
		errexit("Error: internal error: could not parse generated patch to test: %v. Please report this as a bug.\n", err)
	}
	err = p32ps.Validate()
	if err != nil {
		errexit("Error: internal error: could not validate generated patch to test: %v. Please report this as a bug.\n", err)
	}
	err = p32ps.SetEnabled(*patchName, true)
	if err != nil {
		errexit("Error: internal error: could not process generated patch: could not enable patch to test: %v. Please report this as a bug.\n", err)
	}
	p32pt := patchlib.NewPatcher(ibuf)
	err = p32ps.ApplyTo(p32pt)
	if err != nil {
		errexit("Error: internal error: could not apply generated patch to test: %v. Please report this as a bug.\n", err)
	}
	if !bytes.Equal(p32pt.GetBytes(), obuf) {
		errexit("Error: internal error: applied generated patch, wrong output. Please report this as a bug.\n")
	}
	fmt.Printf("--> SUCCESS\n")

	fmt.Printf("\nCONVERTED PATCH:\n")
	fmt.Print(patch)

	os.Exit(0)
}

type replacement struct {
	Offset  int
	Find    []byte
	Replace []byte
}
