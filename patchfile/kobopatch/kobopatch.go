package kobopatch

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"

	"github.com/pgaskin/kobopatch/patchfile"
	"github.com/pgaskin/kobopatch/patchlib"
	"gopkg.in/yaml.v3"
)

type PatchSet struct {
	parsed map[string]*parsedPatch
}

// parsedPatch holds a representation of a PatchNode for use internally. It
// cannot be re-marshaled directly (use the PatchNode and InstructionNode for
// that).
type parsedPatch struct {
	Enabled      bool
	Description  string
	PatchGroups  []string
	Instructions []*parsedInstruction
}

// parsedInstruction holds a representation of a InstructionNode for use internally.
type parsedInstruction struct {
	Index       int
	Line        int
	Instruction PatchableInstruction
}

func init() {
	patchfile.RegisterFormat("kobopatch", Parse)
}

// Parse parses a PatchSet from a buf.
func Parse(buf []byte) (patchfile.PatchSet, error) {
	patchfile.Log("parsing patch file: unmarshaling to map[string]yaml.Node\n")
	var psn map[string]yaml.Node
	if err := yaml.Unmarshal(buf, &psn); err != nil {
		if bytes.Contains(buf, []byte{'\t'}) {
			return nil, fmt.Errorf("patch file contains tabs (it should be indented with spaces, not tabs): %w", err)
		}
		return nil, err
	}

	patchfile.Log("parsing patch file: converting to map[string]*parsedPatch\n")
	ps := PatchSet{map[string]*parsedPatch{}}
	for name, node := range psn {
		patchfile.Log("  unmarshaling patch %#v to PatchNode ([]yaml.Node)\n", name)
		var pn PatchNode
		if err := node.DecodeStrict(&pn); err != nil {
			return nil, fmt.Errorf("line %d: patch %#v: %w", node.Line, name, err)
		}

		patchfile.Log("  converting to []InstructionNode (map[string]yaml.Node)\n")
		ns, err := pn.ToInstructionNodes()
		if err != nil {
			return nil, fmt.Errorf("line %d: patch %#v: %w", node.Line, name, err)
		}

		patchfile.Log("  converting to *parsedPatch\n")
		ps.parsed[name] = &parsedPatch{}
		for i, instNode := range ns {
			patchfile.Log("    unmarshaling instruction %d to Instruction\n", i+1)
			inst, err := instNode.ToInstruction()
			if err != nil {
				return nil, fmt.Errorf("line %d: patch %#v: instruction %d: %w", node.Line, name, i+1, err)
			}

			patchfile.Log("      converting to SingleInstruction...")
			sinst := inst.ToSingleInstruction()
			patchfile.Log("      type=%s\n", reflect.TypeOf(sinst))
			switch sinst.(type) {
			case Enabled:
				ps.parsed[name].Enabled = bool(sinst.(Enabled))
			case Description:
				if ps.parsed[name].Description != "" {
					return nil, fmt.Errorf("patch %#v: line %d: instruction %d: duplicate Description instruction", name, instNode.Line(node.Line), i+1)
				}
				ps.parsed[name].Description = string(sinst.(Description))
			case PatchGroup:
				g := string(sinst.(PatchGroup))
				ps.parsed[name].PatchGroups = append(ps.parsed[name].PatchGroups, g)
			default:
				patchfile.Log("      converting to PatchableInstruction\n")
				if psinst, ok := sinst.(PatchableInstruction); ok {
					ps.parsed[name].Instructions = append(ps.parsed[name].Instructions, &parsedInstruction{i + 1, instNode.Line(node.Line), psinst})
					break
				}
				panic(fmt.Errorf("incomplete implementation (missing implementation of PatchableInstruction) for type %s", reflect.TypeOf(sinst)))
			}
		}
	}
	return &ps, nil
}

// ApplyTo applies a PatchSet to a Patcher.
func (ps *PatchSet) ApplyTo(pt *patchlib.Patcher) error {
	patchfile.Log("validating patch file\n")
	if err := ps.Validate(); err != nil {
		err = fmt.Errorf("invalid patch file: %w", err)
		fmt.Printf("  Error: %v\n", err)
		return err
	}

	patchfile.Log("looping over patches\n")
	for _, name := range ps.SortedNames() {
		patch := ps.parsed[name]
		patchfile.Log("  Patch(%#v) enabled=%t\n", name, patch.Enabled)

		patchfile.Log("    ResetBaseAddress()\n")
		pt.ResetBaseAddress()

		if !patch.Enabled {
			patchfile.Log("    skipping\n")
			fmt.Printf("  SKIP  `%s`\n", name)
			continue
		}

		patchfile.Log("    applying\n")
		fmt.Printf("  APPLY `%s`\n", name)

		patchfile.Log("    looping over instructions\n")
		for _, inst := range patch.Instructions {
			patchfile.Log("      %s index=%d line=%d\n", reflect.TypeOf(inst.Instruction), inst.Index, inst.Line)
			if err := inst.Instruction.ApplyTo(pt, func(format string, a ...interface{}) {
				patchfile.Log("        %s\n", fmt.Sprintf(format, a...))
			}); err != nil {
				err = fmt.Errorf("could not apply patch %#v: line %d: inst %d: %w", name, inst.Line, inst.Index, err)
				patchfile.Log("        %v", err)
				fmt.Printf("    Error: %v\n", err)
				return err
			}
		}
	}

	return nil
}

// SetEnabled sets the Enabled state of a Patch in a PatchSet.
func (ps *PatchSet) SetEnabled(patch string, enabled bool) error {
	if patch, ok := ps.parsed[patch]; ok {
		patch.Enabled = enabled
		return nil
	}
	return fmt.Errorf("no such patch %#v", patch)
}

// SortedNames gets the names of patches sorted alphabetically.
func (ps *PatchSet) SortedNames() []string {
	names := make([]string, len(ps.parsed))
	var i int
	for name := range ps.parsed {
		names[i] = name
		i++
	}
	sort.Strings(names)
	return names
}

// Validate validates the PatchSet.
func (ps *PatchSet) Validate() error {
	usedPatchGroups := map[string]string{}
	for _, name := range ps.SortedNames() {
		patch := ps.parsed[name]

		seenPatchGroups := map[string]bool{}
		for _, g := range patch.PatchGroups {
			if seenPatchGroups[g] {
				return fmt.Errorf("patch %#v: duplicate PatchGroup instruction for PatchGroup %#v", name, g)
			}
			seenPatchGroups[g] = true
			if patch.Enabled {
				if r, ok := usedPatchGroups[g]; ok {
					return fmt.Errorf("patch %#v: more than one patch enabled in PatchGroup %#v (other patch is %#v)", name, g, r)
				}
				usedPatchGroups[g] = name
			}
		}

		if len(patch.Instructions) == 0 {
			return fmt.Errorf("patch %#v: no instructions which modify anything", name)
		}

		for _, inst := range patch.Instructions {
			pfx := fmt.Sprintf("patch %#v: line %d: inst %d", name, inst.Line, inst.Index)
			switch inst.Instruction.(type) {
			case ReplaceBytesNOP:
				if len(inst.Instruction.(ReplaceBytesNOP).Find)%2 != 0 {
					return fmt.Errorf("%s: ReplaceBytesNOP: find must be a multiple of 2 to be replaced with 00 46 (MOV r0, r0)", pfx)
				}
			case ReplaceString:
				if inst.Instruction.(ReplaceString).MustMatchLength {
					if d := len(inst.Instruction.(ReplaceString).Replace) - len(inst.Instruction.(ReplaceString).Find); d < 0 {
						return fmt.Errorf("%s: ReplaceString: replacement string %d chars too short", pfx, -d)
					} else if d > 0 {
						return fmt.Errorf("%s: ReplaceString: replacement string %d chars too long", pfx, d)
					}
				}
			case FindReplaceString:
				if inst.Instruction.(FindReplaceString).MustMatchLength {
					if d := len(inst.Instruction.(FindReplaceString).Replace) - len(inst.Instruction.(FindReplaceString).Find); d < 0 {
						return fmt.Errorf("%s: FindReplaceString: replacement string %d chars too short", pfx, -d)
					} else if d > 0 {
						return fmt.Errorf("%s: FindReplaceString: replacement string %d chars too long", pfx, d)
					}
				}
			case FindZlibHash:
				if len(inst.Instruction.(FindZlibHash)) != 40 {
					return fmt.Errorf("%s: FindZlibHash: hash must be 40 chars long", pfx)
				}
			case ReplaceZlibGroup:
				r := inst.Instruction.(ReplaceZlibGroup)
				if len(r.Replacements) == 0 {
					return fmt.Errorf("%s: ReplaceZlibGroup: no replacements specified", pfx)
				}
				for i, repl := range r.Replacements {
					if repl.Find == "" || repl.Replace == "" {
						return fmt.Errorf("%s: ReplaceZlibGroup: replacement %d: Find and Replace must be set", pfx, i+1)
					}
				}
			}
		}
	}
	return nil
}
