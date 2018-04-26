package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/geek1011/kobopatch/patchlib"
	"github.com/ogier/pflag"
)

var version = "unknown"

func main() {
	patchFile := pflag.StringP("patch-file", "p", "", "File to read patches from (required)")
	inputFile := pflag.StringP("input-file", "i", "", "File to apply patches to (required)")
	outputFile := pflag.StringP("output-file", "o", "", "File to write patched file to")
	help := pflag.BoolP("help", "h", false, "Show this help text")
	pflag.Parse()

	fmt.Printf("patch32lsb %s\n", version)
	fmt.Printf("THIS IS STILL IN PROGESS. DO NOT USE IT ON A REAL DEVICE UNLESS YOU ARE PREPARED TO REIMAGE THE SD CARD.\n\n")

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
			switch strings.ToLower(spl[0]) {
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
				args := strings.Replace(spl[1], " ", "", -1)
				var offset int32
				var find, replace []byte
				_, err := fmt.Sscanf(args, "%x,%x,%x", &offset, &find, &replace)
				checkErrn(i+1, err, "replace_bytes malformed")
				checkErrn(i+1, p.ReplaceBytes(offset, find, replace), "replace_bytes failed")
			case "base_address":
				if !patchEnabled {
					break
				}
				var addr int32
				_, err := fmt.Sscanf(spl[1], "%x", &addr)
				checkErrn(i+1, err, "base_address malformed")
				checkErrn(i+1, p.BaseAddress(addr), "base_address failed")
			case "replace_float":
				if !patchEnabled {
					break
				}
				args := strings.Replace(spl[1], " ", "", -1)
				var offset int32
				var find, replace float64
				_, err := fmt.Sscanf(args, "%x,%f,%f", &offset, &find, &replace)
				checkErrn(i+1, err, "replace_float malformed")
				checkErrn(i+1, p.ReplaceFloat(offset, find, replace), "replace_float failed")
			case "replace_int":
				if !patchEnabled {
					break
				}
				args := strings.Replace(spl[1], " ", "", -1)
				var offset int32
				var find, replace uint8
				_, err := fmt.Sscanf(args, "%x,%d,%d", &offset, &find, &replace)
				checkErrn(i+1, err, "replace_int malformed")
				checkErrn(i+1, p.ReplaceInt(offset, find, replace), "replace_int failed")
			case "find_base_address":
				if !patchEnabled {
					break
				}
				str, err := unescape(spl[1])
				checkErrn(i+1, err, "find_base_address malformed")
				checkErrn(i+1, p.FindBaseAddressString(str), "find_base_address failed")
			case "replace_string":
				if !patchEnabled {
					break
				}
				ab := strings.SplitN(spl[1], ", ", 2)
				if len(ab) != 2 {
					fataln(i+1, "replace_string malformed")
				}
				var offset int32
				if len(ab[0]) == 8 {
					// ugly hack to fix negative offsets
					ab[0] = strings.Replace(ab[0], "FFFFFF", "-", 1)
				}
				_, err := fmt.Sscanf(ab[0], "%x", &offset)
				checkErrn(i+1, err, "replace_string offset malformed")
				var find, replace, leftover string
				leftover = ab[1]
				find, leftover, err = unescapeFirst(leftover)
				checkErrn(i+1, err, "replace_string find malformed")
				leftover = strings.TrimLeft(leftover, ", ")
				replace, leftover, err = unescapeFirst(leftover)
				checkErrn(i+1, err, "replace_string replace malformed")
				if leftover != "" {
					fataln(i+1, "replace_string malformed: extraneous characters after last argument")
				}
				checkErrn(i+1, p.ReplaceString(offset, find, replace), "replace_string failed")
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
	if !(strings.HasPrefix(str, "`") && strings.HasSuffix(str, "`")) || (string(str[len(str)-2]) == `\` && string(str[len(str)-3]) != `\`) {
		return str, errors.New("string not wrapped in backticks")
	}
	str = str[1 : len(str)-1]

	var buf bytes.Buffer
	for {
		if len(str) == 0 {
			break
		}
		switch str[0] {
		case '\\':
			switch str[1] {
			case 'n':
				buf.Write([]byte("\n"))
				str = str[2:]
			case 'r':
				buf.Write([]byte("\r"))
				str = str[2:]
			case 't':
				buf.Write([]byte("\t"))
				str = str[2:]
			case 'v':
				buf.Write([]byte("\v"))
				str = str[2:]
			case '"':
				buf.Write([]byte("\""))
				str = str[2:]
			case '\'':
				buf.Write([]byte("'"))
				str = str[2:]
			case '`':
				buf.Write([]byte("`"))
				str = str[2:]
			case '0':
				buf.Write([]byte("\x00"))
				str = str[2:]
			case '\\':
				buf.Write([]byte("\\"))
				str = str[2:]
			case 'x':
				var b []byte
				_, err := fmt.Sscanf(str[2:4], "%x", &b)
				if err != nil {
					return "", err
				}
				buf.Write(b)
				str = str[4:]
			default:
				return "", errors.New("unknown escape " + string(str[1]))
			}
		default:
			buf.Write([]byte{str[0]})
			str = str[1:]
		}
	}
	return string(buf.Bytes()), nil
}

func unescapeFirst(str string) (string, string, error) {
	// TODO: make more efficient
	for i := 2; i <= len(str); i++ {
		nstr, err := unescape(str[:i])
		if err == nil {
			return nstr, str[i:], nil
		}
	}
	return "", "", errors.New("could not find valid string")
}
