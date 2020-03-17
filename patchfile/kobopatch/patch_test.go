package kobopatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInstructionNodeToInstruction(t *testing.T) {
	tc := func(msg, y string, i *Instruction, equality bool, eerr error, remarshal bool) {
		var n InstructionNode
		if err := yaml.Unmarshal([]byte(y), &n); err != nil {
			panic(err)
		}
		t.Run(msg, func(t *testing.T) {
			ai, err := n.ToInstruction()
			if a, b := fmt.Sprint(eerr), fmt.Sprint(err); a != b {
				t.Errorf("unexpected error in ToInstruction: expected %#v, got %#v", a, b)
			}
			if eerr != nil {
				return
			}
			bufa, bufb := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
			json.NewEncoder(bufa).Encode(*i)
			json.NewEncoder(bufb).Encode(ai)
			if equality != reflect.DeepEqual(bufa.String(), bufb.String()) {
				t.Errorf("wrong result of ToInstruction: expected %s, got %s", bufa.String(), bufb.String())
			}
			if remarshal {
				t.Run("Marshal", func(t *testing.T) {
					buf, err := yaml.Marshal(ai)
					if err != nil {
						t.Errorf("unexpected error re-marshaling node: %v", err)
					}
					buf = bytes.TrimRight(buf, "\n")
					if !bytes.Equal(buf, []byte(y)) {
						t.Errorf("wrong re-marshaled bytes: expected orig %#v, got %#v", y, string(buf))
					}
				})
			}
		})
	}
	tc("None", ``, nil, true, errors.New("expected instruction, got nothing"), false)
	tc("Unknown", `Unknown: true`, nil, true, errors.New("line 1: unknown instruction type \"Unknown\""), false)
	d := int32(1)
	a := BaseAddress(FlexAbsOffset{Offset: &d, Inline: true})
	tc("TooMany", `{BaseAddress: 1, FindBaseAddressString: "test"}`, &Instruction{BaseAddress: &a}, false, errors.New("line 1: multiple types found in instruction, maybe you forgot a '-'"), false)
	tc("ValueEqual", `BaseAddress: 1`, &Instruction{BaseAddress: &a}, true, nil, false)
	tc("ValueNotEqual", `BaseAddress: 0`, &Instruction{}, false, nil, false)
	tc("StructEqual", `FindReplaceString: {Find: "test", Replace: "test"}`, &Instruction{FindReplaceString: &FindReplaceString{Find: "test", Replace: "test"}}, true, nil, false)
	tc("StructNotEqual", `FindReplaceString: {Find: "test", Replace: "test"}`, &Instruction{FindReplaceString: &FindReplaceString{Find: "test", Replace: ""}}, false, nil, false)
	tc("StructExtra", `FindReplaceString: {Find: "test", Replace: "test", Extra: "asd"}`, &Instruction{FindReplaceString: &FindReplaceString{Find: "test", Replace: "text"}}, false, errors.New("line 1: error decoding instruction: yaml: unmarshal errors:\n  line 1: field Extra not found in type kobopatch.FindReplaceString"), false)

	e := "Test"
	tc("FlexAbsOffset/Inline/BaseAddress", `BaseAddress: 1`, &Instruction{BaseAddress: &BaseAddress{Offset: &d, Inline: true}}, true, nil, true)
	tc("FlexAbsOffset/Offset/BaseAddress", `BaseAddress: {Offset: 1}`, &Instruction{BaseAddress: &BaseAddress{Offset: &d}}, true, nil, true)
	tc("FlexAbsOffset/Sym/BaseAddress", `BaseAddress: {Sym: Test}`, &Instruction{BaseAddress: &BaseAddress{Sym: &e}}, true, nil, true)
	tc("FlexAbsOffset/SymPLT/BaseAddress", `BaseAddress: {SymPLT: Test}`, &Instruction{BaseAddress: &BaseAddress{SymPLT: &e}}, true, nil, true)
	tc("FlexAbsOffset/SymPLTTail/BaseAddress", `BaseAddress: {SymPLTTail: Test}`, &Instruction{BaseAddress: &BaseAddress{SymPLTTail: &e}}, true, nil, true)
	tc("FlexAbsOffset/Inline/ReplaceBytesBase", `ReplaceBytes: {Base: 1}`, &Instruction{ReplaceBytes: &ReplaceBytes{Base: &FlexAbsOffset{Offset: &d, Inline: true}}}, true, nil, false)
	tc("FlexAbsOffset/Offset/ReplaceBytesBase", `ReplaceBytes: {Base: {Offset: 1}}`, &Instruction{ReplaceBytes: &ReplaceBytes{Base: &FlexAbsOffset{Offset: &d}}}, true, nil, false)
	tc("FlexAbsOffset/Sym/ReplaceBytesBase", `ReplaceBytes: {Base: {Sym: Test}}`, &Instruction{ReplaceBytes: &ReplaceBytes{Base: &FlexAbsOffset{Sym: &e}}}, true, nil, false)
	tc("FlexAbsOffset/SymPLT/ReplaceBytesBase", `ReplaceBytes: {Base: {SymPLT: Test}}`, &Instruction{ReplaceBytes: &ReplaceBytes{Base: &FlexAbsOffset{SymPLT: &e}}}, true, nil, false)
	tc("FlexAbsOffset/SymPLTTail/ReplaceBytesBase", `ReplaceBytes: {Base: {SymPLTTail: Test}}`, &Instruction{ReplaceBytes: &ReplaceBytes{Base: &FlexAbsOffset{SymPLTTail: &e}}}, true, nil, false)
	// TODO: more FlexAbsOffset tests?
}
