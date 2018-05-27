package kobopatch

import (
	"fmt"
	"strings"

	"github.com/geek1011/kobopatch/kobopatch/formats"
	"github.com/geek1011/kobopatch/patchlib"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
)

// PatchSet represents a series of patches.
type PatchSet map[string]patch

type patch []instruction
type instruction struct {
	Enabled               *bool   `yaml:"Enabled,omitempty"`
	Description           *string `yaml:"Description,omitempty"`
	PatchGroup            *string `yaml:"PatchGroup,omitempty"`
	BaseAddress           *int32  `yaml:"BaseAddress,omitempty"`
	FindBaseAddressHex    *string `yaml:"FindBaseAddressHex,omitempty"`
	FindBaseAddressString *string `yaml:"FindBaseAddressString,omitempty"`
	FindReplaceString     *struct {
		Find    string `yaml:"Find,omitempty"`
		Replace string `yaml:"Replace,omitempty"`
	} `yaml:"FindReplaceString,omitempty"`
	ReplaceString *struct {
		Offset  int32  `yaml:"Offset,omitempty"`
		Find    string `yaml:"Find,omitempty"`
		Replace string `yaml:"Replace,omitempty"`
	} `yaml:"ReplaceString,omitempty"`
	ReplaceInt *struct {
		Offset  int32 `yaml:"Offset,omitempty"`
		Find    uint8 `yaml:"Find,omitempty"`
		Replace uint8 `yaml:"Replace,omitempty"`
	} `yaml:"ReplaceInt,omitempty"`
	ReplaceFloat *struct {
		Offset  int32   `yaml:"Offset,omitempty"`
		Find    float64 `yaml:"Find,omitempty"`
		Replace float64 `yaml:"Replace,omitempty"`
	} `yaml:"ReplaceFloat,omitempty"`
	ReplaceBytes *struct {
		Offset   int32   `yaml:"Offset,omitempty"`
		FindH    *string `yaml:"FindH,omitempty"`
		ReplaceH *string `yaml:"ReplaceH,omitempty"`
		Find     []byte  `yaml:"Find,omitempty"`
		Replace  []byte  `yaml:"Replace,omitempty"`
	} `yaml:"ReplaceBytes,omitempty"`
}

// Parse parses a PatchSet from a buf.
func Parse(buf []byte) (formats.PatchSet, error) {
	formats.Log("parsing patch file\n")
	ps := &PatchSet{}
	if err := yaml.UnmarshalStrict(buf, &ps); err != nil {
		return nil, errors.Wrap(err, "error parsing patch file")
	}

	formats.Log("parsing patch file: expanding shorthand hex values\n")
	for n := range *ps {
		for i := range (*ps)[n] {
			if (*ps)[n][i].ReplaceBytes != nil {
				if ((*ps)[n][i].ReplaceBytes).FindH != nil {
					hex := *((*ps)[n][i].ReplaceBytes).FindH
					_, err := fmt.Sscanf(
						strings.Replace(hex, " ", "", -1),
						"%x\n",
						&((*ps)[n][i].ReplaceBytes).Find,
					)
					if err != nil {
						formats.Log("  error decoding hex `%s`: %v\n", hex, err)
						return nil, errors.Errorf("error parsing patch file: error expanding shorthand hex `%s`", hex)
					}
					formats.Log("  decoded hex `%s` to `%v`\n", hex, ((*ps)[n][i].ReplaceBytes).Find)
				}
				if ((*ps)[n][i].ReplaceBytes).ReplaceH != nil {
					hex := *((*ps)[n][i].ReplaceBytes).ReplaceH
					_, err := fmt.Sscanf(
						strings.Replace(hex, " ", "", -1),
						"%x\n",
						&((*ps)[n][i].ReplaceBytes).Replace,
					)
					if err != nil {
						formats.Log("  error decoding hex `%s`: %v\n", hex, err)
						return nil, errors.Errorf("error parsing patch file: error expanding shorthand hex `%s`", hex)
					}
					formats.Log("  decoded hex `%s` to `%v`\n", hex, ((*ps)[n][i].ReplaceBytes).Replace)
				}
			}
		}
	}

	return ps, nil
}

// Validate validates the PatchSet.
func (ps *PatchSet) Validate() error {
	enabledPatchGroups := map[string]bool{}
	for n, p := range *ps {
		ec := 0
		e := false
		pgc := 0
		pg := ""
		dc := 0

		rbc := 0
		roc := 0
		fbsc := 0

		for _, i := range p {
			ic := 0
			if i.Enabled != nil {
				ec++
				e = *i.Enabled
				ic++
			}
			if i.Description != nil {
				dc++
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
			if i.FindBaseAddressString != nil {
				ic++
				fbsc++
			}
			if i.FindBaseAddressHex != nil {
				ic++
			}
			if i.ReplaceBytes != nil {
				ic++
				rbc++
			}
			if i.ReplaceFloat != nil {
				ic++
				roc++
			}
			if i.ReplaceInt != nil {
				ic++
				roc++
			}
			if i.ReplaceString != nil {
				ic++
				roc++
			}
			if i.FindReplaceString != nil {
				ic++
				roc++
			}
			formats.Log("  ic:%d\n", ic)
			if ic < 1 {
				return errors.Errorf("internal error while validating `%s` (you should report this as a bug)", n)
			}
			if ic > 1 {
				return errors.Errorf("more than one instruction per bullet in patch `%s` (you might be missing a -)", n)
			}
		}
		formats.Log("  ec:%d, e:%t, pgc:%d, pg:%s, dc:%d, rbc:%d, roc: %d, fbsc:%d\n", ec, e, pgc, pg, dc, rbc, roc, fbsc)
		if ec < 1 {
			return errors.Errorf("no `Enabled` option in `%s`", n)
		} else if ec > 1 {
			return errors.Errorf("more than one `Enabled` option in `%s`", n)
		}
		if dc > 1 {
			return errors.Errorf("more than one `Description` option in `%s` (use comments to describe individual lines)", n)
		}
		if pgc > 1 {
			return errors.Errorf("more than one `PatchGroup` option in `%s`", n)
		}
		if pg != "" && e {
			if _, ok := enabledPatchGroups[pg]; ok {
				return errors.Errorf("more than one patch enabled in PatchGroup `%s`", pg)
			}
			enabledPatchGroups[pg] = true
		}
		if roc == 0 && rbc > 0 && fbsc > 0 {
			return errors.Errorf("use FindBaseAddressHex for hex replacements because FindBaseAddressString will lose control characters (patch `%s`)", n)
		}
	}
	formats.Log("  enabledPatchGroups:%v\n", enabledPatchGroups)
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
			case i.Enabled != nil || i.PatchGroup != nil || i.Description != nil:
				formats.Log("  skipping non-instruction Enabled(), PatchGroup() or Description()\n")
				// Skip non-instructions
				err = nil
			case i.BaseAddress != nil:
				formats.Log("  BaseAddress(%#v)\n", *i.BaseAddress)
				err = pt.BaseAddress(*i.BaseAddress)
			case i.FindBaseAddressHex != nil:
				formats.Log("  FindBaseAddressHex(%#v)\n", *i.FindBaseAddressHex)
				buf := []byte{}
				_, err = fmt.Sscanf(strings.Replace(*i.FindBaseAddressHex, " ", "", -1), "%x\n", &buf)
				if err != nil {
					err = errors.Errorf("FindBaseAddresHex: invalid hex string")
					break
				}
				err = pt.FindBaseAddress(buf)
			case i.FindBaseAddressString != nil:
				formats.Log("  FindBaseAddressString(%#v) | hex:%x\n", *i.FindBaseAddressString, []byte(*i.FindBaseAddressString))
				err = pt.FindBaseAddressString(*i.FindBaseAddressString)
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
			case i.FindReplaceString != nil:
				r := *i.FindReplaceString
				formats.Log("  FindReplaceString(%#v, %#v)\n", r.Find, r.Replace)
				formats.Log("    FindBaseAddressString(%#v)\n", r.Find)
				err = pt.FindBaseAddressString(r.Find)
				if err != nil {
					err = errors.Wrap(err, "FindReplaceString")
					break
				}
				formats.Log("    ReplaceString(0, %#v, %#v)\n", r.Find, r.Replace)
				err = pt.ReplaceString(0, r.Find, r.Replace)
				if err != nil {
					err = errors.Wrap(err, "FindReplaceString")
					break
				}
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

func init() {
	formats.RegisterFormat("kobopatch", Parse)
}
