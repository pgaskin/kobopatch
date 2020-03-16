package kobopatch

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/geek1011/kobopatch/patchlib"
	"gopkg.in/yaml.v3"
)

type Patch []*Instruction

type PatchNode []yaml.Node

func (p *PatchNode) ToInstructionNodes() ([]InstructionNode, error) {
	n := make([]InstructionNode, len(*p))
	for i, t := range *p {
		if err := t.DecodeStrict(&n[i]); err != nil {
			return n, err
		}
	}
	return n, nil
}

func (p *PatchNode) ToPatch() (Patch, error) {
	n := make(Patch, len(*p))
	for i, t := range *p {
		if err := t.DecodeStrict(&n[i]); err != nil {
			return n, err
		}
	}
	return n, nil
}

type Instruction struct {
	Enabled               *Enabled               `yaml:"Enabled,omitempty"`
	Description           *Description           `yaml:"Description,omitempty"`
	PatchGroup            *PatchGroup            `yaml:"PatchGroup,omitempty"`
	BaseAddress           *BaseAddress           `yaml:"BaseAddress,omitempty"`
	FindBaseAddressHex    *FindBaseAddressHex    `yaml:"FindBaseAddressHex,omitempty"`
	FindBaseAddressString *FindBaseAddressString `yaml:"FindBaseAddressString,omitempty"`
	FindBaseAddressSymbol *FindBaseAddressSymbol `yaml:"FindBaseAddressSymbol,omitempty"`
	FindZlib              *FindZlib              `yaml:"FindZlib,omitempty"`
	FindZlibHash          *FindZlibHash          `yaml:"FindZlibHash,omitempty"`
	FindReplaceString     *FindReplaceString     `yaml:"FindReplaceString,omitempty"`
	ReplaceString         *ReplaceString         `yaml:"ReplaceString,omitempty"`
	ReplaceInt            *ReplaceInt            `yaml:"ReplaceInt,omitempty"`
	ReplaceFloat          *ReplaceFloat          `yaml:"ReplaceFloat,omitempty"`
	ReplaceBytes          *ReplaceBytes          `yaml:"ReplaceBytes,omitempty"`
	ReplaceBytesAtSymbol  *ReplaceBytesAtSymbol  `yaml:"ReplaceBytesAtSymbol,omitempty"`
	ReplaceBytesNOP       *ReplaceBytesNOP       `yaml:"ReplaceBytesNOP,omitempty"`
	ReplaceZlib           *ReplaceZlib           `yaml:"ReplaceZlib,omitempty"`
	ReplaceZlibGroup      *ReplaceZlibGroup      `yaml:"ReplaceZlibGroup,omitempty"`
	ReplaceBLX            *ReplaceBLX            `yaml:"ReplaceBLX,omitempty"`
}

type InstructionNode map[string]yaml.Node

func (i InstructionNode) ToInstruction() (*Instruction, error) {
	if len(i) == 0 {
		return nil, fmt.Errorf("expected instruction, got nothing")
	}
	var found bool
	var n Instruction
	for name, node := range i {
		if found {
			return nil, fmt.Errorf("line %d: multiple types found in instruction, maybe you forgot a '-'", node.Line)
		} else if field := reflect.ValueOf(&n).Elem().FieldByName(name); !field.IsValid() {
			return nil, fmt.Errorf("line %d: unknown instruction type %#v", node.Line, name)
		} else if err := node.DecodeStrict(field.Addr().Interface()); err != nil {
			return nil, fmt.Errorf("line %d: error decoding instruction: %w", node.Line, err)
		} else {
			found = true
		}
	}
	return &n, nil
}

func (i InstructionNode) Line(def int) int {
	for _, node := range i {
		return node.Line
	}
	return def
}

func (i Instruction) ToSingleInstruction() interface{} {
	iv := reflect.ValueOf(i)
	for i := 0; i < iv.NumField(); i++ {
		if !iv.Field(i).IsNil() {
			return iv.Field(i).Elem().Interface()
		}
	}
	return nil
}

type Enabled bool
type Description string
type PatchGroup string

type PatchableInstruction interface {
	ApplyTo(*patchlib.Patcher, func(string, ...interface{})) error
}

type BaseAddress int32
type FindBaseAddressHex string
type FindBaseAddressString string
type FindBaseAddressSymbol string
type FindZlib string
type FindZlibHash string

func (b BaseAddress) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("BaseAddress(%#v)", b)
	return pt.BaseAddress(int32(b))
}

func (b FindBaseAddressHex) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("FindBaseAddressHex(%#v)", b)
	var buf []byte
	_, err := fmt.Sscanf(strings.ReplaceAll(string(b), " ", ""), "%x\n", &buf)
	if err != nil {
		return fmt.Errorf("FindBaseAddressHex: error parsing hex: %w", err)
	}
	if err := pt.FindBaseAddress(buf); err != nil {
		return fmt.Errorf("FindBaseAddressHex: %w", err)
	}
	return nil
}

func (b FindBaseAddressString) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("FindBaseAddressString(%#v) | hex:%x", b, []byte(b))
	return pt.FindBaseAddressString(string(b))
}

func (b FindBaseAddressSymbol) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("FindBaseAddressSymbol(%#v)", b)
	return pt.FindBaseAddressSymbol(string(b))
}

func (b FindZlib) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("FindZlib(%#v) | hex:%x", b, []byte(b))
	return pt.FindZlib(string(b))
}

func (b FindZlibHash) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("FindZlibHash(%#v) | hex:%x", b, []byte(b))
	return pt.FindZlibHash(string(b))
}

type FindReplaceString struct {
	Find            string `yaml:"Find"`
	Replace         string `yaml:"Replace"`
	MustMatchLength bool   `yaml:"MustMatchLength,omitempty"`
}

type ReplaceString struct {
	Offset          int32  `yaml:"Offset,omitempty"`
	Find            string `yaml:"Find"`
	Replace         string `yaml:"Replace"`
	MustMatchLength bool   `yaml:"MustMatchLength,omitempty"`
}

type ReplaceInt struct {
	Offset  int32 `yaml:"Offset,omitempty"`
	Find    uint8 `yaml:"Find"`
	Replace uint8 `yaml:"Replace"`
}

type ReplaceFloat struct {
	Offset  int32   `yaml:"Offset,omitempty"`
	Find    float64 `yaml:"Find"`
	Replace float64 `yaml:"Replace"`
}

type ReplaceBytes struct {
	Offset   int32   `yaml:"Offset,omitempty"`
	FindH    *string `yaml:"FindH,omitempty"`
	ReplaceH *string `yaml:"ReplaceH,omitempty"`
	FindBLX  *uint32 `yaml:"FindBLX,omitempty"`
	Find     []byte  `yaml:"Find,omitempty"`
	Replace  []byte  `yaml:"Replace,omitempty"`
}

type ReplaceBytesAtSymbol struct {
	Symbol   string  `yaml:"Symbol,omitempty"`
	Offset   int32   `yaml:"Offset,omitempty"`
	FindH    *string `yaml:"FindH,omitempty"`
	ReplaceH *string `yaml:"ReplaceH,omitempty"`
	FindBLX  *uint32 `yaml:"FindBLX,omitempty"`
	Find     []byte  `yaml:"Find,omitempty"`
	Replace  []byte  `yaml:"Replace,omitempty"`
}

type ReplaceBytesNOP struct {
	Offset  int32   `yaml:"Offset,omitempty"`
	FindH   *string `yaml:"FindH,omitempty"`
	FindBLX *uint32 `yaml:"FindBLX,omitempty"`
	Find    []byte  `yaml:"Find,omitempty"`
}

type ReplaceZlib struct {
	Offset  int32  `yaml:"Offset,omitempty"`
	Find    string `yaml:"Find"`
	Replace string `yaml:"Replace"`
}

type ReplaceZlibGroup struct {
	Offset       int32 `yaml:"Offset,omitempty"`
	Replacements []struct {
		Find    string `yaml:"Find"`
		Replace string `yaml:"Replace"`
	} `yaml:"Replacements"`
}

type ReplaceBLX struct {
	Offset  int32  `yaml:"Offset,omitempty"`
	Find    uint32 `yaml:"Find"`
	Replace uint32 `yaml:"Replace"`
}

func (r FindReplaceString) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("FindReplaceString(%#v, %#v)", r.Find, r.Replace)
	log("  FindBaseAddressString(%#v)", r.Find)
	if err := pt.FindBaseAddressString(r.Find); err != nil {
		return fmt.Errorf("FindReplaceString: %w", err)
	}
	log("  ReplaceString(0, %#v, %#v)", r.Find, r.Replace)
	if err := pt.ReplaceString(0, r.Find, r.Replace); err != nil {
		return fmt.Errorf("FindReplaceString: %w", err)
	}
	return nil
}

func (r ReplaceString) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("ReplaceString(%#v, %#v, %#v)", r.Offset, r.Find, r.Replace)
	return pt.ReplaceString(r.Offset, r.Find, r.Replace)
}

func (r ReplaceInt) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("ReplaceInt(%#v, %#v, %#v)", r.Offset, r.Find, r.Replace)
	return pt.ReplaceInt(r.Offset, r.Find, r.Replace)
}

func (r ReplaceFloat) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("ReplaceFloat(%#v, %#v, %#v)", r.Offset, r.Find, r.Replace)
	return pt.ReplaceFloat(r.Offset, r.Find, r.Replace)
}

func (r ReplaceBytes) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	if replaced, err := expandHex(r.FindH, &r.Find); err != nil {
		log("ReplaceBytes.FindH -> %v", err)
		return err
	} else if replaced {
		log("ReplaceBytes.FindH -> Expand <%s> to set ReplaceBytesNOP.Find to <%x>", *r.FindH, r.Find)
	}
	if replaced, err := expandHex(r.ReplaceH, &r.Replace); err != nil {
		log("ReplaceBytes.ReplaceH -> %v", err)
		return err
	} else if replaced {
		log("ReplaceBytes.ReplaceH -> Expand <%s> to set ReplaceBytesNOP.Replace to <%x>", *r.ReplaceH, r.Replace)
	}
	if r.FindBLX != nil {
		r.Find = patchlib.AsmBLX(uint32(pt.GetCur()+r.Offset), *r.FindBLX)
		log("ReplaceBytes.FindBLX -> Set ReplaceBytes.Find to BLX(0x%X, 0x%X) -> %X", pt.GetCur()+r.Offset, *r.FindBLX, r.Find)
	}
	log("ReplaceBytes(%#v, %#v, %#v)", r.Offset, r.Find, r.Replace)
	return pt.ReplaceBytes(r.Offset, r.Find, r.Replace)
}

func (r ReplaceBytesAtSymbol) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	if replaced, err := expandHex(r.FindH, &r.Find); err != nil {
		log("ReplaceBytesAtSymbol.FindH -> %v", err)
		return err
	} else if replaced {
		log("ReplaceBytesAtSymbol.FindH -> Expand <%s> to set ReplaceBytesAtSymbol.Find to <%x>", *r.FindH, r.Find)
	}
	if replaced, err := expandHex(r.ReplaceH, &r.Replace); err != nil {
		log("ReplaceBytesAtSymbol.ReplaceH -> %v", err)
		return err
	} else if replaced {
		log("ReplaceBytesAtSymbol.ReplaceH -> Expand <%s> to set ReplaceBytesAtSymbol.Replace to <%x>", *r.ReplaceH, r.Replace)
	}

	log("  ReplaceBytesAtSymbol(%#v, %#v, %#v, %#v)", r.Symbol, r.Offset, r.Find, r.Replace)
	log("    FindBaseAddressSymbol(%#v) -> ", r.Symbol)
	if err := pt.FindBaseAddressSymbol(r.Symbol); err != nil {
		return fmt.Errorf("ReplaceBytesAtSymbol: %w", err)
	}
	log("      0x%06x", pt.GetCur())
	if r.FindBLX != nil {
		r.Find = patchlib.AsmBLX(uint32(pt.GetCur()+r.Offset), *r.FindBLX)
		log("    ReplaceBytesAtSymbol.FindBLX -> Set ReplaceBytesAtSymbol.Find to BLX(0x%X, 0x%X) -> %X", pt.GetCur()+r.Offset, *r.FindBLX, r.Find)
	}
	log("    ReplaceBytes(%#v, %#v, %#v)", r.Offset, r.Find, r.Replace)
	if err := pt.ReplaceBytes(r.Offset, r.Find, r.Replace); err != nil {
		return fmt.Errorf("ReplaceBytesAtSymbol: %w", err)
	}
	return nil
}

func (r ReplaceBytesNOP) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	if replaced, err := expandHex(r.FindH, &r.Find); err != nil {
		log("ReplaceBytesNOP.FindH -> %v", err)
		return err
	} else if replaced {
		log("ReplaceBytesNOP.FindH -> Expand <%s> to set ReplaceBytesNOP.Find to <%x>", *r.FindH, r.Find)
	}
	if r.FindBLX != nil {
		r.Find = patchlib.AsmBLX(uint32(pt.GetCur()+r.Offset), *r.FindBLX)
		log("ReplaceBytesNOP.FindBLX -> Set ReplaceBytesNOP.Find to BLX(0x%X, 0x%X) -> %X", pt.GetCur()+r.Offset, *r.FindBLX, r.Find)
	}
	log("ReplaceBytesNOP(%#v, %#v)", r.Offset, r.Find)
	return pt.ReplaceBytesNOP(r.Offset, r.Find)
}

func (r ReplaceZlib) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("ReplaceZlib(%#v, %#v, %#v)", r.Offset, r.Find, r.Replace)
	return pt.ReplaceZlib(r.Offset, r.Find, r.Replace)
}

func (r ReplaceZlibGroup) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("ReplaceZlibGroup(%#v, %#v)", r.Offset, r.Replacements)
	rs := []patchlib.Replacement{}
	for _, rr := range r.Replacements {
		rs = append(rs, patchlib.Replacement{Find: rr.Find, Replace: rr.Replace})
	}
	return pt.ReplaceZlibGroup(r.Offset, rs)
}

func (r ReplaceBLX) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("ReplaceBLX(%#v, %#v, %#v)", r.Offset, r.Find, r.Replace)
	return pt.ReplaceBLX(r.Offset, r.Find, r.Replace)
}

func expandHex(in *string, out *[]byte) (bool, error) {
	if in == nil {
		return false, nil
	}
	if _, err := fmt.Sscanf(strings.ReplaceAll(*in, " ", ""), "%x\n", out); err != nil {
		return true, fmt.Errorf("error expanding shorthand hex `%s`: %w", *in, err)
	}
	return true, nil
}
