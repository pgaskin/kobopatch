package kobopatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func TestInstructionNodeToInstruction(t *testing.T) {
	tc := func(msg, y string, i *Instruction, equality bool, eerr error) {
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
		})
	}
	tc("None", ``, nil, true, errors.New("expected instruction, got nothing"))
	tc("Unknown", `Unknown: true`, nil, true, errors.New("line 1: unknown instruction type \"Unknown\""))
	a := BaseAddress(1)
	tc("TooMany", `BaseAddress: 1`+"\n"+`FindBaseAddressString: "test"`, &Instruction{BaseAddress: &a}, false, errors.New("line 2: multiple types found in instruction, maybe you forgot a '-'"))
	tc("ValueEqual", `BaseAddress: 1`, &Instruction{BaseAddress: &a}, true, nil)
	tc("ValueNotEqual", `BaseAddress: 0`, &Instruction{}, false, nil)
	tc("StructEqual", `FindReplaceString: {Find: "test", Replace: "test"}`, &Instruction{FindReplaceString: &FindReplaceString{Find: "test", Replace: "test"}}, true, nil)
	tc("StructNotEqual", `FindReplaceString: {Find: "test", Replace: "test"}`, &Instruction{FindReplaceString: &FindReplaceString{Find: "test", Replace: ""}}, false, nil)
	tc("StructExtra", `FindReplaceString: {Find: "test", Replace: "test", Extra: "asd"}`, &Instruction{FindReplaceString: &FindReplaceString{Find: "test", Replace: "text"}}, false, errors.New("line 1: error decoding instruction: yaml: unmarshal errors:\n  line 1: field Extra not found in type kobopatch.FindReplaceString"))
}
