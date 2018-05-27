package patch32lsb

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/geek1011/kobopatch/kobopatch/formats"
	"github.com/geek1011/kobopatch/patchlib"
)

// PatchSet represents a series of patches.
type PatchSet map[string]patch

type patch []instruction
type instruction struct {
	Enabled         *bool
	PatchGroup      *string
	BaseAddress     *int32
	Comment         *string
	FindBaseAddress *string
	ReplaceBytes    *struct {
		Offset  int32
		Find    []byte
		Replace []byte
	}
	ReplaceFloat *struct {
		Offset  int32
		Find    float64
		Replace float64
	}
	ReplaceInt *struct {
		Offset  int32
		Find    uint8
		Replace uint8
	}
	ReplaceString *struct {
		Offset  int32
		Find    string
		Replace string
	}
}

// Parse parses a PatchSet from a buf.
func Parse(buf []byte) (formats.PatchSet, error) {
	// TODO: make less hacky, make cleaner, add logs

	ps := PatchSet{}
	var patchName string
	var inPatch bool
	curPatch := patch{}
	eqRegexp := regexp.MustCompile(" +?= +?")
	for i, l := range strings.Split(strings.Replace(string(buf), "\r\n", "\n", -1), "\n") {
		l = strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(l, "#"), l == "":
			c := strings.TrimLeft(l, "# ")
			if strings.HasPrefix(c, "patch_group") {
				return nil, errors.Errorf("line %d: patch_group should not be a comment", i+1)
			}
			curPatch = append(curPatch, instruction{Comment: &c})
			break
		case strings.ToLower(l) == "<patch>":
			if inPatch {
				return nil, errors.Errorf("line %d: unexpected <Patch> (already in patch)", i+1)
			}
			curPatch = patch{}
			inPatch = true
			break
		case strings.ToLower(l) == "</patch>":
			if !inPatch {
				return nil, errors.Errorf("line %d: unexpected </Patch> (not in patch)", i+1)
			}
			if patchName == "" {
				return nil, errors.Errorf("line %d: no patch_name for patch", i+1)
			}
			if _, ok := ps[patchName]; ok {
				return nil, errors.Errorf("line %d: duplicate patch with name '%s'", i+1, patchName)
			}
			ps[patchName] = curPatch[:]
			inPatch = false
			break
		case !eqRegexp.MatchString(l):
			return nil, errors.Errorf("line %d: bad instruction: no equals sign", i+1)
		case eqRegexp.MatchString(l):
			spl := eqRegexp.Split(l, 2)
			switch strings.ToLower(spl[0]) {
			case "patch_name":
				var err error
				patchName, err = unescape(spl[1])
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: error unescaping patch_name", i+1)
				}
			case "patch_group":
				g, err := unescape(spl[1])
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: error unescaping patch_group", i+1)
				}
				curPatch = append(curPatch, instruction{PatchGroup: &g})
			case "patch_enable":
				if patchName == "" {
					return nil, errors.Errorf("line %d: patch_enable set before patch_name", i+1)
				}
				switch spl[1] {
				case "`yes`":
					e := true
					curPatch = append(curPatch, instruction{Enabled: &e})
				case "`no`":
					e := false
					curPatch = append(curPatch, instruction{Enabled: &e})
				default:
					return nil, errors.Errorf("line %d: unexpected patch_enable value '%s' (should be yes or no)", i+1, spl[1])
				}
			case "replace_bytes":
				args := strings.Replace(spl[1], " ", "", -1)
				var offset int32
				var find, replace []byte
				_, err := fmt.Sscanf(args, "%x,%x,%x", &offset, &find, &replace)
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: replace_bytes malformed", i+1)
				}
				curPatch = append(curPatch, instruction{ReplaceBytes: &struct {
					Offset  int32
					Find    []byte
					Replace []byte
				}{Offset: offset, Find: find, Replace: replace}})
			case "base_address":
				var addr int32
				_, err := fmt.Sscanf(spl[1], "%x", &addr)
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: base_address malformed", i+1)
				}
				curPatch = append(curPatch, instruction{BaseAddress: &addr})
			case "replace_float":
				args := strings.Replace(spl[1], " ", "", -1)
				var offset int32
				var find, replace float64
				_, err := fmt.Sscanf(args, "%x,%f,%f", &offset, &find, &replace)
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: replace_float malformed", i+1)
				}
				curPatch = append(curPatch, instruction{ReplaceFloat: &struct {
					Offset  int32
					Find    float64
					Replace float64
				}{Offset: offset, Find: find, Replace: replace}})
			case "replace_int":
				args := strings.Replace(spl[1], " ", "", -1)
				var offset int32
				var find, replace uint8
				_, err := fmt.Sscanf(args, "%x,%d,%d", &offset, &find, &replace)
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: replace_int malformed", i+1)
				}
				curPatch = append(curPatch, instruction{ReplaceInt: &struct {
					Offset  int32
					Find    uint8
					Replace uint8
				}{Offset: offset, Find: find, Replace: replace}})
			case "find_base_address":
				str, err := unescape(spl[1])
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: find_base_address malformed", i+1)
				}
				curPatch = append(curPatch, instruction{FindBaseAddress: &str})
			case "replace_string":
				ab := strings.SplitN(spl[1], ", ", 2)
				if len(ab) != 2 {
					return nil, errors.Errorf("line %d: replace_string malformed", i+1)
				}
				var offset int32
				if len(ab[0]) == 8 {
					// ugly hack to fix negative offsets
					ab[0] = strings.Replace(ab[0], "FFFFFF", "-", 1)
				}
				_, err := fmt.Sscanf(ab[0], "%x", &offset)
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: replace_string offset malformed", i+1)
				}
				var find, replace, leftover string
				leftover = ab[1]
				find, leftover, err = unescapeFirst(leftover)
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: replace_string find malformed", i+1)
				}
				leftover = strings.TrimLeft(leftover, ", ")
				replace, leftover, err = unescapeFirst(leftover)
				if err != nil {
					return nil, errors.Wrapf(err, "line %d: replace_string replace malformed", i+1)
				}
				if leftover != "" {
					return nil, errors.Errorf("line %d: replace_string malformed: extraneous characters after last argument", i+1)
				}
				curPatch = append(curPatch, instruction{ReplaceString: &struct {
					Offset  int32
					Find    string
					Replace string
				}{Offset: offset, Find: find, Replace: replace}})
			default:
				return nil, errors.Errorf("line %d: unexpected instruction: %s", i+1, spl[0])
			}
		default:
			return nil, errors.Errorf("line %d: unexpected statement: %s", i+1, l)
		}
	}
	return &ps, nil
}

// Validate validates the PatchSet.
func (ps *PatchSet) Validate() error {
	enabledPatchGroups := map[string]bool{}
	for n, p := range *ps {
		pgc := 0
		ec := 0
		e := false
		pg := ""

		for _, i := range p {
			ic := 0
			if i.Enabled != nil {
				ec++
				e = *i.Enabled
				ic++
			}
			if i.PatchGroup != nil {
				pgc++
				pg = *i.PatchGroup
				ic++
			}
			if i.BaseAddress != nil {
				ic++
			}
			if i.Comment != nil {
				ic++
			}
			if i.FindBaseAddress != nil {
				ic++
			}
			if i.ReplaceBytes != nil {
				ic++
			}
			if i.ReplaceFloat != nil {
				ic++
			}
			if i.ReplaceInt != nil {
				ic++
			}
			if i.ReplaceString != nil {
				ic++
			}
			if ic != 1 {
				return errors.Errorf("internal error (you should report this): ic > 1, '%#v'", i)
			}
		}
		if ec != 1 {
			return errors.Errorf("you must have exactly 1 patch_enable option in each patch (%s)", n)
		}
		if pgc > 1 {
			return errors.Errorf("you must have at most 1 patch_group option in each patch (%s)", n)
		}
		if pg != "" && e {
			if _, ok := enabledPatchGroups[pg]; ok {
				return errors.Errorf("more than one patch enabled in patch_group '%s'", pg)
			}
			enabledPatchGroups[pg] = true
		}
	}
	return nil
}

// ApplyTo applies a PatchSet to a Patcher.
func (ps *PatchSet) ApplyTo(pt *patchlib.Patcher) error {
	formats.Log("validating patch file\n")
	err := ps.Validate()
	if err != nil {
		err = errors.Wrap(err, "invalid patch file")
		fmt.Printf("  Error: %v\n", err)
		return err
	}

	formats.Log("looping over patches\n")
	num, total := 0, len(*ps)
	for n, p := range *ps {
		var err error
		num++
		formats.Log("  ResetBaseAddress()\n")
		pt.ResetBaseAddress()

		enabled := false
		for _, i := range p {
			if i.Enabled != nil && *i.Enabled {
				enabled = *i.Enabled
				break
			}
		}
		formats.Log("  Enabled: %t\n", enabled)

		if !enabled {
			formats.Log("  skipping patch `%s`\n", n)
			fmt.Printf("  [%d/%d] Skipping disabled patch `%s`\n", num, total, n)
			continue
		}

		formats.Log("  applying patch `%s`\n", n)
		fmt.Printf("  [%d/%d] Applying patch `%s`\n", num, total, n)

		formats.Log("looping over instructions\n")
		for _, i := range p {
			switch {
			case i.Enabled != nil || i.PatchGroup != nil || i.Comment != nil:
				formats.Log("  skipping non-instruction Enabled(), PatchGroup() or Comment()\n")
				// Skip non-instructions
				err = nil
			case i.BaseAddress != nil:
				formats.Log("  BaseAddress(%#v)\n", *i.BaseAddress)
				err = pt.BaseAddress(*i.BaseAddress)
			case i.FindBaseAddress != nil:
				formats.Log("  FindBaseAddressString(%#v) | hex:%x\n", *i.FindBaseAddress, []byte(*i.FindBaseAddress))
				err = pt.FindBaseAddressString(*i.FindBaseAddress)
			case i.ReplaceBytes != nil:
				r := *i.ReplaceBytes
				formats.Log("  ReplaceBytes(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceBytes(r.Offset, r.Find, r.Replace)
			case i.ReplaceFloat != nil:
				r := *i.ReplaceFloat
				formats.Log("  ReplaceFloat(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceFloat(r.Offset, r.Find, r.Replace)
			case i.ReplaceInt != nil:
				r := *i.ReplaceInt
				formats.Log("  ReplaceInt(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceInt(r.Offset, r.Find, r.Replace)
			case i.ReplaceString != nil:
				r := *i.ReplaceString
				formats.Log("  ReplaceString(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceString(r.Offset, r.Find, r.Replace)
			default:
				formats.Log("  invalid instruction: %#v\n", i)
				err = errors.Errorf("invalid instruction: %#v", i)
			}

			if err != nil {
				formats.Log("could not apply patch: %v\n", err)
				fmt.Printf("    Error: could not apply patch: %v\n", err)
				return err
			}
		}
	}

	return nil
}

// SetEnabled sets the Enabled state of a Patch in a PatchSet.
func (ps *PatchSet) SetEnabled(patch string, enabled bool) error {
	for n, p := range *ps {
		if n != patch {
			continue
		}
		for _, i := range p {
			if i.Enabled != nil {
				i.Enabled = &enabled
				return nil
			}
		}
		return errors.Errorf("could not set enabled state of '%s' to %t: no Enabled instruction in patch", patch, enabled)
	}
	return errors.Errorf("could not set enabled state of '%s' to %t: no such patch", patch, enabled)
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

func init() {
	formats.RegisterFormat("patch32lsb", Parse)
}
