package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"unicode"

	"github.com/geek1011/kobopatch/patchfile/patch32lsb"
	wordwrap "github.com/mitchellh/go-wordwrap"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "p32lsb2kobopatch is an experimental tool to convert patch32lsb style patches to kobopatch ones. It currently does not work with the old-style zlib patches.")
		fmt.Fprintln(os.Stderr, "Usage: p32lsb2kobopatch PATCH_FILE > OUTPUT_FILE.yaml")
		os.Exit(1)
	}

	ibuf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	tipf, err := patch32lsb.Parse(ibuf)
	if err != nil {
		panic(err)
	}

	err = tipf.Validate()
	if err != nil {
		panic(err)
	}

	ipf := reflect.ValueOf(tipf).Interface().(*patch32lsb.PatchSet)

	for patchName, instructions := range *ipf {
		fmt.Printf("%s: \n", patchName)
		fmt.Printf("  - Enabled: no\n")

		for _, inst := range instructions {
			if inst.PatchGroup != nil {
				fmt.Printf("  - PatchGroup: %s\n", *inst.PatchGroup)
			}
		}

		var desc string
		in := false
		for _, inst := range instructions {
			if inst.Comment != nil {
				if in && strings.TrimSpace(desc) != "" && strings.Trim(*inst.Comment, "# \t") == "" {
					break // Skip after a blank comment
				}
				if strings.Contains(*inst.Comment, "Multi-version patch") {
					continue
				}
				in = true
				desc += strings.TrimSpace(*inst.Comment) + " "
			} else if in {
				break // Skip after first group of comments
			}
		}
		if strings.TrimSpace(desc) != "" {
			desc = wordwrap.WrapString(desc, 70)
			fmt.Printf("  - Description: |\n%s\n", setIndent(desc, 6))
		}

		for i, inst := range instructions {
			switch {
			case inst.BaseAddress != nil:
				fmt.Printf("  - BaseAddress: 0x%X\n", *inst.BaseAddress)
			case inst.FindBaseAddress != nil:
				if !isASCII(*inst.FindBaseAddress) {
					fmt.Printf("  - FindBaseAddressHex: % X\n", *inst.FindBaseAddress)
					continue
				}
				if ni := instructions[i+1]; ni.ReplaceString != nil {
					if (*ni.ReplaceString).Find == *inst.FindBaseAddress {
						if len((*ni.ReplaceString).Find) > 60 || len((*ni.ReplaceString).Replace) > 60 {
							fmt.Printf("  - FindReplaceString:\n      Find: %s\n      Replace: %s\n", escapeString((*ni.ReplaceString).Find), escapeString((*ni.ReplaceString).Replace))
						} else {
							fmt.Printf("  - FindReplaceString: {Find: %s, Replace: %s}\n", escapeString((*ni.ReplaceString).Find), escapeString((*ni.ReplaceString).Replace))
						}
						instructions[i+1].ReplaceString = nil
						continue
					}
				}
				fmt.Printf("  - FindBaseAddressString: %s\n", escapeString(*inst.FindBaseAddress))
			case inst.ReplaceBytes != nil:
				fmt.Printf("  - ReplaceBytes: {Offset: %d, FindH: % X, ReplaceH: % X}\n", (*inst.ReplaceBytes).Offset, (*inst.ReplaceBytes).Find, (*inst.ReplaceBytes).Replace)
				// panic("todo")
			case inst.ReplaceFloat != nil:
				fmt.Printf("  - ReplaceFloat: {Offset: %d, Find: %f, Replace: %f}\n", (*inst.ReplaceFloat).Offset, (*inst.ReplaceFloat).Find, (*inst.ReplaceFloat).Replace)
			case inst.ReplaceInt != nil:
				fmt.Printf("  - ReplaceInt: {Offset: %d, Find: %d, Replace: %d}\n", (*inst.ReplaceInt).Offset, (*inst.ReplaceInt).Find, (*inst.ReplaceInt).Replace)
			case inst.ReplaceString != nil:
				if len((*inst.ReplaceString).Find) > 60 || len((*inst.ReplaceString).Replace) > 60 {
					fmt.Printf("  - ReplaceString:\n      Offset: %d\n      Find: %s\n      Replace: %s\n", (*inst.ReplaceString).Offset, escapeString((*inst.ReplaceString).Find), escapeString((*inst.ReplaceString).Replace))
				} else {
					fmt.Printf("  - ReplaceString: {Offset: %d, Find: %s, Replace: %s}\n", (*inst.ReplaceString).Offset, escapeString((*inst.ReplaceString).Find), escapeString((*inst.ReplaceString).Replace))
				}
			case inst.FindZlib != nil:
				fmt.Printf("  - FindZlib: %s\n", escapeString(*inst.FindZlib))
				continue
			case inst.FindZlibHash != nil:
				fmt.Printf("  - FindZlibHash: %s\n", escapeString(*inst.FindZlibHash))
				continue
			case inst.ReplaceZlib != nil:
				if len((*inst.ReplaceZlib).Find) > 60 || len((*inst.ReplaceZlib).Replace) > 60 {
					fmt.Printf("  - ReplaceZlib:\n      Offset: %d\n      Find: %s\n      Replace: %s\n", (*inst.ReplaceZlib).Offset, escapeString((*inst.ReplaceZlib).Find), escapeString((*inst.ReplaceZlib).Replace))
				} else {
					fmt.Printf("  - ReplaceZlib: {Offset: %d, Find: %s, Replace: %s}\n", (*inst.ReplaceZlib).Offset, escapeString((*inst.ReplaceZlib).Find), escapeString((*inst.ReplaceZlib).Replace))
				}
			default:
				continue
			}
		}

		_ = instructions
		fmt.Printf("\n")
	}
}

func setIndent(str string, i int) string {
	var out string
	for _, s := range strings.Split(str, "\n") {
		if strings.TrimSpace(s) == "" {
			continue
		}
		out += strings.Repeat(" ", i) + strings.TrimLeft(s, " \t") + "\n"
	}
	return strings.TrimRight(out, "\n")
}

func escapeString(str string) string {
	buf, err := marshal(str)
	if err != nil {
		panic(err)
	}
	nstr := strings.TrimSpace(string(buf))
	nstr = strings.Replace(nstr, `\u0000`, `\0`, -1)
	nstr = strings.Replace(nstr, `\u0002`, `\x02`, -1)
	nstr = strings.Replace(nstr, `\u0004`, `\x04`, -1)
	nstr = strings.Replace(nstr, `\u0006`, `\x06`, -1)
	nstr = strings.Replace(nstr, `\u0008`, `\x08`, -1)
	nstr = strings.Replace(nstr, `\u000c`, `\x0c`, -1)
	nstr = strings.Replace(nstr, `\u000e`, `\x0e`, -1)
	return nstr
}

func marshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return buffer.Bytes(), err
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII || !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
