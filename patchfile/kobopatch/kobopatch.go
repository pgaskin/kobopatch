// Package kobopatch reads kobopatch style patches.
package kobopatch

import (
	"fmt"
	"strings"

	"github.com/geek1011/kobopatch/patchfile"
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
	FindBaseAddressSymbol *string `yaml:"FindBaseAddressSymbol,omitempty"`
	FindZlib              *string `yaml:"FindZlib,omitempty"`
	FindZlibHash          *string `yaml:"FindZlibHash,omitempty"`
	FindReplaceString     *struct {
		Find            string `yaml:"Find,omitempty"`
		Replace         string `yaml:"Replace,omitempty"`
		MustMatchLength bool   `yaml:"MustMatchLength,omitempty"`
	} `yaml:"FindReplaceString,omitempty"`
	ReplaceString *struct {
		Offset          int32  `yaml:"Offset,omitempty"`
		Find            string `yaml:"Find,omitempty"`
		Replace         string `yaml:"Replace,omitempty"`
		MustMatchLength bool   `yaml:"MustMatchLength,omitempty"`
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
		FindBLX  *uint32 `yaml:"FindBLX,omitempty"`
		Find     []byte  `yaml:"Find,omitempty"`
		Replace  []byte  `yaml:"Replace,omitempty"`
	} `yaml:"ReplaceBytes,omitempty"`
	ReplaceBytesAtSymbol *struct {
		Symbol   string  `yaml:"Symbol,omitempty"`
		Offset   int32   `yaml:"Offset,omitempty"`
		FindH    *string `yaml:"FindH,omitempty"`
		ReplaceH *string `yaml:"ReplaceH,omitempty"`
		FindBLX  *uint32 `yaml:"FindBLX,omitempty"`
		Find     []byte  `yaml:"Find,omitempty"`
		Replace  []byte  `yaml:"Replace,omitempty"`
	} `yaml:"ReplaceBytesAtSymbol,omitempty"`
	ReplaceBytesNOP *struct {
		Offset  int32   `yaml:"Offset,omitempty"`
		FindH   *string `yaml:"FindH,omitempty"`
		FindBLX *uint32 `yaml:"FindBLX,omitempty"`
		Find    []byte  `yaml:"Find,omitempty"`
	} `yaml:"ReplaceBytesNOP,omitempty"`
	ReplaceZlib *struct {
		Offset  int32  `yaml:"Offset,omitempty"`
		Find    string `yaml:"Find,omitempty"`
		Replace string `yaml:"Replace,omitempty"`
	} `yaml:"ReplaceZlib,omitempty"`
	ReplaceZlibGroup *struct {
		Offset       int32 `yaml:"Offset,omitempty"`
		Replacements []struct {
			Find    string `yaml:"Find,omitempty"`
			Replace string `yaml:"Replace,omitempty"`
		} `yaml:"Replacements,omitempty"`
	} `yaml:"ReplaceZlibGroup,omitempty"`
	ReplaceBLX *struct {
		Offset  int32  `yaml:"Offset,omitempty"`
		Find    uint32 `yaml:"Find,omitempty"`
		Replace uint32 `yaml:"Replace,omitempty"`
	} `yaml:"ReplaceBLX,omitempty"`
}

// Parse parses a PatchSet from a buf.
func Parse(buf []byte) (patchfile.PatchSet, error) {
	patchfile.Log("parsing patch file\n")
	ps := &PatchSet{}
	if err := yaml.UnmarshalStrict(buf, &ps); err != nil {
		return nil, errors.Wrap(err, "error parsing patch file")
	}

	patchfile.Log("parsing patch file: expanding shorthand hex values\n")
	for n := range *ps {
		for i := range (*ps)[n] {
			if (*ps)[n][i].ReplaceBytesNOP != nil {
				if ((*ps)[n][i].ReplaceBytesNOP).FindH != nil {
					hex := *((*ps)[n][i].ReplaceBytesNOP).FindH
					_, err := fmt.Sscanf(
						strings.Replace(hex, " ", "", -1),
						"%x\n",
						&((*ps)[n][i].ReplaceBytesNOP).Find,
					)
					if err != nil {
						patchfile.Log("  error decoding hex `%s`: %v\n", hex, err)
						return nil, errors.Errorf("error parsing patch file: error expanding shorthand hex `%s`", hex)
					}
					patchfile.Log("  decoded hex `%s` to `%v`\n", hex, ((*ps)[n][i].ReplaceBytesNOP).Find)
				}
			}
			if (*ps)[n][i].ReplaceBytes != nil {
				if ((*ps)[n][i].ReplaceBytes).FindH != nil {
					hex := *((*ps)[n][i].ReplaceBytes).FindH
					_, err := fmt.Sscanf(
						strings.Replace(hex, " ", "", -1),
						"%x\n",
						&((*ps)[n][i].ReplaceBytes).Find,
					)
					if err != nil {
						patchfile.Log("  error decoding hex `%s`: %v\n", hex, err)
						return nil, errors.Errorf("error parsing patch file: error expanding shorthand hex `%s`", hex)
					}
					patchfile.Log("  decoded hex `%s` to `%v`\n", hex, ((*ps)[n][i].ReplaceBytes).Find)
				}
				if ((*ps)[n][i].ReplaceBytes).ReplaceH != nil {
					hex := *((*ps)[n][i].ReplaceBytes).ReplaceH
					_, err := fmt.Sscanf(
						strings.Replace(hex, " ", "", -1),
						"%x\n",
						&((*ps)[n][i].ReplaceBytes).Replace,
					)
					if err != nil {
						patchfile.Log("  error decoding hex `%s`: %v\n", hex, err)
						return nil, errors.Errorf("error parsing patch file: error expanding shorthand hex `%s`", hex)
					}
					patchfile.Log("  decoded hex `%s` to `%v`\n", hex, ((*ps)[n][i].ReplaceBytes).Replace)
				}
			}
			if (*ps)[n][i].ReplaceBytesAtSymbol != nil {
				if ((*ps)[n][i].ReplaceBytesAtSymbol).FindH != nil {
					hex := *((*ps)[n][i].ReplaceBytesAtSymbol).FindH
					_, err := fmt.Sscanf(
						strings.Replace(hex, " ", "", -1),
						"%x\n",
						&((*ps)[n][i].ReplaceBytesAtSymbol).Find,
					)
					if err != nil {
						patchfile.Log("  error decoding hex `%s`: %v\n", hex, err)
						return nil, errors.Errorf("error parsing patch file: error expanding shorthand hex `%s`", hex)
					}
					patchfile.Log("  decoded hex `%s` to `%v`\n", hex, ((*ps)[n][i].ReplaceBytesAtSymbol).Find)
				}
				if ((*ps)[n][i].ReplaceBytesAtSymbol).ReplaceH != nil {
					hex := *((*ps)[n][i].ReplaceBytesAtSymbol).ReplaceH
					_, err := fmt.Sscanf(
						strings.Replace(hex, " ", "", -1),
						"%x\n",
						&((*ps)[n][i].ReplaceBytesAtSymbol).Replace,
					)
					if err != nil {
						patchfile.Log("  error decoding hex `%s`: %v\n", hex, err)
						return nil, errors.Errorf("error parsing patch file: error expanding shorthand hex `%s`", hex)
					}
					patchfile.Log("  decoded hex `%s` to `%v`\n", hex, ((*ps)[n][i].ReplaceBytesAtSymbol).Replace)
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

		for instn, i := range p {
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
			if i.FindBaseAddressSymbol != nil {
				ic++
			}
			if i.FindBaseAddressHex != nil {
				ic++
			}
			if i.ReplaceBytes != nil {
				ic++
				rbc++
			}
			if i.ReplaceBytesAtSymbol != nil {
				ic++
				rbc++
			}
			if i.ReplaceBytesNOP != nil {
				ic++
				rbc++
				if len(i.ReplaceBytesNOP.Find)%2 != 0 {
					return errors.Errorf("i%d: find must be a multiple of 2 to be replaced with 00 46 (MOV r0, r0)", instn+1)
				}
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
				if i.ReplaceString.MustMatchLength {
					if len(i.ReplaceString.Find) < len(i.ReplaceString.Replace) {
						return errors.Errorf("i%d: replacement string must not be shorter in `%s`", instn+1, n)
					} else if len(i.ReplaceString.Find) > len(i.ReplaceString.Replace) {
						return errors.Errorf("i%d: replacement string too long in `%s`", instn+1, n)
					}
				}
			}
			if i.FindReplaceString != nil {
				ic++
				roc++
				if i.FindReplaceString.MustMatchLength {
					if len(i.FindReplaceString.Find) != len(i.FindReplaceString.Replace) {
						return errors.Errorf("i%d: length of strings must match (and not be shorter) in `%s`", instn+1, n)
					}
				}
			}
			if i.FindZlib != nil {
				ic++
				roc++
			}
			if i.FindZlibHash != nil {
				ic++
				roc++
				if len(*i.FindZlibHash) != 40 {
					return errors.Errorf("i%d: hash must be 40 chars in FindZlibHash in `%s`", instn+1, n)
				}
			}
			if i.ReplaceZlib != nil {
				ic++
				roc++
			}
			if i.ReplaceZlibGroup != nil {
				ic++
				roc++
				if len((*i.ReplaceZlibGroup).Replacements) < 1 {
					return errors.Errorf("i%d: no replacements in ReplaceZlibGroup instruction", instn+1)
				}
				for _, r := range (*i.ReplaceZlibGroup).Replacements {
					if r.Find == "" {
						return errors.Errorf("i%d: find is required in replacement in ReplaceZlibGroup instruction", instn+1)
					} else if r.Replace == "" {
						return errors.Errorf("i%d: replace is required in replacement in ReplaceZlibGroup instruction", instn+1)
					}
				}
			}
			if i.ReplaceBLX != nil {
				ic++
				roc++
			}
			if ic < 1 {
				return errors.Errorf("i%d: internal error while validating `%s` (you should report this as a bug)", instn+1, n)
			}
			if ic > 1 {
				return errors.Errorf("i%d: more than one instruction per bullet in patch `%s` (you might be missing a -)", instn+1, n)
			}
		}
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
	patchfile.Log("  enabledPatchGroups:%v\n", enabledPatchGroups)
	return nil
}

// ApplyTo applies a PatchSet to a Patcher.
func (ps *PatchSet) ApplyTo(pt *patchlib.Patcher) error {
	patchfile.Log("validating patch file\n")
	err := ps.Validate()
	if err != nil {
		err = errors.Wrap(err, "invalid patch file")
		fmt.Printf("  Error: %v\n", err)
		return err
	}

	patchfile.Log("looping over patches\n")
	num := 0
	for n, p := range *ps {
		var err error
		num++
		patchfile.Log("  ResetBaseAddress()\n")
		pt.ResetBaseAddress()

		enabled := false
		for _, i := range p {
			if i.Enabled != nil && *i.Enabled {
				enabled = *i.Enabled
				break
			}
		}
		patchfile.Log("  Enabled: %t\n", enabled)

		if !enabled {
			patchfile.Log("  skipping patch `%s`\n", n)
			fmt.Printf("  SKIP  `%s`\n", n)
			continue
		}

		patchfile.Log("  applying patch `%s`\n", n)
		fmt.Printf("  APPLY `%s`\n", n)

		patchfile.Log("looping over instructions\n")
		for instn, i := range p {
			switch {
			case i.Enabled != nil || i.PatchGroup != nil || i.Description != nil:
				patchfile.Log("  skipping non-instruction Enabled(), PatchGroup() or Description()\n")
				// Skip non-instructions
				err = nil
			case i.BaseAddress != nil:
				patchfile.Log("  BaseAddress(%#v)\n", *i.BaseAddress)
				err = pt.BaseAddress(*i.BaseAddress)
			case i.FindBaseAddressHex != nil:
				patchfile.Log("  FindBaseAddressHex(%#v)\n", *i.FindBaseAddressHex)
				buf := []byte{}
				_, err = fmt.Sscanf(strings.Replace(*i.FindBaseAddressHex, " ", "", -1), "%x\n", &buf)
				if err != nil {
					err = errors.Errorf("FindBaseAddressHex: invalid hex string")
					break
				}
				err = pt.FindBaseAddress(buf)
			case i.FindBaseAddressString != nil:
				patchfile.Log("  FindBaseAddressString(%#v) | hex:%x\n", *i.FindBaseAddressString, []byte(*i.FindBaseAddressString))
				err = pt.FindBaseAddressString(*i.FindBaseAddressString)
			case i.FindBaseAddressSymbol != nil:
				patchfile.Log("  FindBaseAddressSymbol(%#v) | hex:%x\n", *i.FindBaseAddressSymbol, []byte(*i.FindBaseAddressSymbol))
				err = pt.FindBaseAddressSymbol(*i.FindBaseAddressSymbol)
			case i.ReplaceBytes != nil:
				r := *i.ReplaceBytes
				if r.FindBLX != nil {
					r.Find = patchlib.BLX(uint32(pt.GetCur()+r.Offset), *r.FindBLX)
					patchfile.Log("  ReplaceBytes.FindBLX -> Set ReplaceBytes.Find to BLX(0x%X, 0x%X) -> %X", pt.GetCur()+r.Offset, *r.FindBLX, r.Find)
				}
				patchfile.Log("  ReplaceBytes(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceBytes(r.Offset, r.Find, r.Replace)
			case i.ReplaceBytesAtSymbol != nil:
				r := *i.ReplaceBytesAtSymbol
				patchfile.Log("  ReplaceBytesAtSymbol(%#v, %#v, %#v, %#v)\n", r.Symbol, r.Offset, r.Find, r.Replace)
				patchfile.Log("    FindBaseAddressSymbol(%#v) -> ", r.Symbol)
				err = pt.FindBaseAddressSymbol(r.Symbol)
				if err != nil {
					err = errors.Wrap(err, "ReplaceBytesAtSymbol")
					break
				}
				patchfile.Log("0x%06x\n", pt.GetCur())
				if r.FindBLX != nil {
					r.Find = patchlib.BLX(uint32(pt.GetCur()+r.Offset), *r.FindBLX)
					patchfile.Log("  ReplaceBytesAtSymbol.FindBLX -> Set ReplaceBytesAtSymbol.Find to BLX(0x%X, 0x%X) -> %X", pt.GetCur()+r.Offset, *r.FindBLX, r.Find)
				}
				patchfile.Log("cur=0x%06x off=0x%x bytes=%x find=%x replace=%x\n", pt.GetCur(), r.Offset, pt.GetBytes()[pt.GetCur()+r.Offset:pt.GetCur()+r.Offset+4], r.Find, r.Replace)
				patchfile.Log("    ReplaceBytes(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceBytes(r.Offset, r.Find, r.Replace)
				if err != nil {
					err = errors.Wrap(err, "ReplaceBytesAtSymbol")
					break
				}
			case i.ReplaceBytesNOP != nil:
				r := *i.ReplaceBytesNOP
				if r.FindBLX != nil {
					r.Find = patchlib.BLX(uint32(pt.GetCur()+r.Offset), *r.FindBLX)
					patchfile.Log("  ReplaceBytesNOP.FindBLX -> Set ReplaceBytesNOP.Find to BLX(0x%X, 0x%X) -> %X", pt.GetCur()+r.Offset, *r.FindBLX, r.Find)
				}
				patchfile.Log("  ReplaceBytesNOP(%#v, %#v)\n", r.Offset, r.Find)
				err = pt.ReplaceBytesNOP(r.Offset, r.Find)
			case i.ReplaceFloat != nil:
				r := *i.ReplaceFloat
				patchfile.Log("  ReplaceFloat(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceFloat(r.Offset, r.Find, r.Replace)
			case i.ReplaceInt != nil:
				r := *i.ReplaceInt
				patchfile.Log("  ReplaceInt(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceInt(r.Offset, r.Find, r.Replace)
			case i.ReplaceString != nil:
				r := *i.ReplaceString
				patchfile.Log("  ReplaceString(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceString(r.Offset, r.Find, r.Replace)
			case i.FindReplaceString != nil:
				r := *i.FindReplaceString
				patchfile.Log("  FindReplaceString(%#v, %#v)\n", r.Find, r.Replace)
				patchfile.Log("    FindBaseAddressString(%#v)\n", r.Find)
				err = pt.FindBaseAddressString(r.Find)
				if err != nil {
					err = errors.Wrap(err, "FindReplaceString")
					break
				}
				patchfile.Log("    ReplaceString(0, %#v, %#v)\n", r.Find, r.Replace)
				err = pt.ReplaceString(0, r.Find, r.Replace)
				if err != nil {
					err = errors.Wrap(err, "FindReplaceString")
					break
				}
			case i.FindZlib != nil:
				patchfile.Log("  FindZlib(%#v) | hex:%x\n", *i.FindZlib, []byte(*i.FindZlib))
				err = pt.FindZlib(*i.FindZlib)
			case i.FindZlibHash != nil:
				patchfile.Log("  FindZlibHash(%#v) | hex:%x\n", *i.FindZlibHash, []byte(*i.FindZlibHash))
				err = pt.FindZlibHash(*i.FindZlibHash)
			case i.ReplaceZlib != nil:
				r := *i.ReplaceZlib
				patchfile.Log("  ReplaceZlib(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceZlib(r.Offset, r.Find, r.Replace)
			case i.ReplaceZlibGroup != nil:
				r := *i.ReplaceZlibGroup
				patchfile.Log("  ReplaceZlibGroup(%#v, %#v)\n", r.Offset, r.Replacements)
				rs := []patchlib.Replacement{}
				for _, rr := range r.Replacements {
					rs = append(rs, patchlib.Replacement{Find: rr.Find, Replace: rr.Replace})
				}
				err = pt.ReplaceZlibGroup(r.Offset, rs)
			case i.ReplaceBLX != nil:
				r := *i.ReplaceBLX
				patchfile.Log("  ReplaceBLX(%#v, %#v, %#v)\n", r.Offset, r.Find, r.Replace)
				err = pt.ReplaceBLX(r.Offset, r.Find, r.Replace)
			default:
				patchfile.Log("  invalid instruction: %#v\n", i)
				err = errors.Errorf("invalid instruction: %#v", i)
			}

			if err != nil {
				patchfile.Log("could not apply patch: i%d: %v\n", instn+1, err)
				fmt.Printf("    Error: i%d: could not apply patch: %v\n", instn+1, err)
				return err
			}
		}
	}
	return nil
}

// SetEnabled sets the Enabled state of a Patch in a PatchSet.
func (ps *PatchSet) SetEnabled(patch string, enabled bool) error {
	for n := range *ps {
		if n != patch {
			continue
		}
		for i := range (*ps)[n] {
			if (*ps)[n][i].Enabled != nil {
				*(*ps)[n][i].Enabled = enabled
				return nil
			}
		}
		return errors.Errorf("could not set enabled state of '%s' to %t: no Enabled instruction in patch", patch, enabled)
	}
	if enabled {
		return errors.Errorf("could not set enabled state of '%s' to %t: no such patch", patch, enabled)
	}
	return nil
}

func init() {
	patchfile.RegisterFormat("kobopatch", Parse)
}
