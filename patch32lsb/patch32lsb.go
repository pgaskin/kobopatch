package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/geek1011/kobopatch/patchlib"
	"github.com/ogier/pflag"
)

func main() {
	patchFile := pflag.StringP("patch-file", "p", "", "File to read patches from (required)")
	inputFile := pflag.StringP("input-file", "i", "", "File to apply patches to (required)")
	outputFile := pflag.StringP("output-file", "o", "", "File to write patched file to")
	help := pflag.BoolP("help", "h", false, "Show this help text")
	pflag.Parse()

	if *help || *inputFile == "" || *patchFile == "" {
		pflag.Usage()
		os.Exit(1)
	}

	pbuf, err := ioutil.ReadFile(*patchFile)
	checkErr(err, "Error reading patch file")

	ibuf, err := ioutil.ReadFile(*inputFile)
	checkErr(err, "Error reading input file")

	p := patchlib.NewPatcher(ibuf)

	var inPatch, patchEnabled bool
	var patchName string
	eqRegexp := regexp.MustCompile(" +?= +?")
	for i, l := range strings.Split(strings.Replace(string(pbuf), "\r\n", "\n", -1), "\n") {
		l = strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(l, "#"), l == "":
			break
		case strings.ToLower(l) == "<patch>":
			if !inPatch {
				p.ResetBaseAddress()
				inPatch = true
				patchEnabled = false
				patchName = ""
				break
			}
			fataln(i+1, "Unexpected <Patch>: Already in patch")
		case strings.ToLower(l) == "</patch>":
			if inPatch {
				inPatch = false
				break
			}
			fataln(i+1, "Unexpected </Patch>: Not in patch")
		case !eqRegexp.MatchString(l):
			fataln(i+1, "Could not find equals sign in instruction")
		case eqRegexp.MatchString(l):
			spl := eqRegexp.Split(l, 2)
			switch spl[0] {
			case "patch_name":
				patchName, err = unescape(spl[1])
				checkErrn(i+1, err, "Error unescaping patch_name")
			case "patch_enable":
				if patchName == "" {
					fataln(i+1, "patch_enabled set before patch_name")
				}
				switch spl[1] {
				case "`yes`":
					fmt.Printf("Applying patch: %s\n", patchName)
					patchEnabled = true
				case "`no`":
					fmt.Printf("Ignoring disabled patch: %s\n", patchName)
					patchEnabled = false
				default:
					fataln(i+1, "unexpected patch_enabled value")
				}
			case "replace_bytes":
				if !patchEnabled {
					break
				}
				fmt.Println("    replace_bytes not implemented")
			case "base_address":
				if !patchEnabled {
					break
				}
				fmt.Println("    base_address not implemented")
			case "replace_float":
				if !patchEnabled {
					break
				}
				fmt.Println("    base_address not implemented")
			case "replace_int":
				if !patchEnabled {
					break
				}
				fmt.Println("    base_address not implemented")
			case "find_base_address":
				if !patchEnabled {
					break
				}
				fmt.Println("    find_base_address not implemented")
			case "replace_string":
				if !patchEnabled {
					break
				}
				fmt.Println("    find_base_address not implemented")
			default:
				fataln(i+1, "Unexpected instruction: "+spl[0])
			}
		default:
			fataln(i+1, "Unexpected statement: "+l)
		}
	}

	err = ioutil.WriteFile(*outputFile, p.GetBytes(), 0644)
	checkErr(err, "Error writing output file")
}

func fataln(n int, msg string) {
	fmt.Fprintf(os.Stderr, "    Fatal: Line %d: %s\n", n, msg)
	os.Exit(1)
}

func checkErr(err error, msg string) {
	if err == nil {
		return
	}
	if msg != "" {
		fmt.Fprintf(os.Stderr, "Fatal: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
	}
	os.Exit(1)
}

func checkErrn(n int, err error, msg string) {
	if err == nil {
		return
	}
	if msg != "" {
		fmt.Fprintf(os.Stderr, "    Fatal: Line %d: %s: %v\n", n, msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "    Fatal: Line %d:  %v\n", n, err)
	}
	os.Exit(1)
}

func unescape(str string) (string, error) {
	if strings.HasPrefix(str, "`") || strings.HasPrefix(str, "'") {
		str = `"` + str[1:]
	}
	if strings.HasSuffix(str, "`") || strings.HasSuffix(str, "'") {
		str = str[:len(str)-1] + `"`
	}
	str = strings.NewReplacer(
		`\0`, `\x00`,
		`\'`, `'`,
		"\\`", "`",
	).Replace(str)
	str, err := strconv.Unquote(str)
	return str, err
}
