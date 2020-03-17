package kobopatch

import (
	"encoding/hex"
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
	BaseAddress           *BaseAddress           `yaml:"BaseAddress,omitempty,flow"`
	FindBaseAddressHex    *FindBaseAddressHex    `yaml:"FindBaseAddressHex,omitempty"`
	FindBaseAddressString *FindBaseAddressString `yaml:"FindBaseAddressString,omitempty"`
	FindZlib              *FindZlib              `yaml:"FindZlib,omitempty"`
	FindZlibHash          *FindZlibHash          `yaml:"FindZlibHash,omitempty"`
	FindReplaceString     *FindReplaceString     `yaml:"FindReplaceString,omitempty"`
	ReplaceString         *ReplaceString         `yaml:"ReplaceString,omitempty"`
	ReplaceInt            *ReplaceInt            `yaml:"ReplaceInt,omitempty,flow"`
	ReplaceFloat          *ReplaceFloat          `yaml:"ReplaceFloat,omitempty,flow"`
	ReplaceBytes          *ReplaceBytes          `yaml:"ReplaceBytes,omitempty"`
	ReplaceZlib           *ReplaceZlib           `yaml:"ReplaceZlib,omitempty"`
	ReplaceZlibGroup      *ReplaceZlibGroup      `yaml:"ReplaceZlibGroup,omitempty"`
	FindBaseAddressSymbol *FindBaseAddressSymbol `yaml:"FindBaseAddressSymbol,omitempty"` // Deprecated: Use BaseAddress instead.
	ReplaceBytesAtSymbol  *ReplaceBytesAtSymbol  `yaml:"ReplaceBytesAtSymbol,omitempty"`  // Deprecated: Use ReplaceBytes.Base instead.
	ReplaceBytesNOP       *ReplaceBytesNOP       `yaml:"ReplaceBytesNOP,omitempty"`       // Deprecated: Use ReplaceBytes.ReplaceNOP instead.
	ReplaceBLX            *ReplaceBLX            `yaml:"ReplaceBLX,omitempty"`            // Deprecated: Use ReplaceBytes.FindInstBLX and ReplaceBytes.ReplaceInstBLX instead.
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

// FlexAbsOffset allows specifying an absolute offset with either a direct
// integer (absolute offset - Offset), string (normal symbol - Sym), or a field.
type FlexAbsOffset struct {
	Offset     *int32  `yaml:"Offset,omitempty"` // can be specified in place of this object
	Sym        *string `yaml:"Sym,omitempty"`
	SymPLT     *string `yaml:"SymPLT,omitempty"`
	SymPLTTail *string `yaml:"SymPLTTail,omitempty"`
	Inline     bool    `yaml:"-"`             // whether the Offset/Sym was inline
	Rel        *int32  `yaml:"Rel,omitempty"` // optional, gets added to the absolute offset found
}

func (f *FlexAbsOffset) UnmarshalYAML(n *yaml.Node) error {
	*f = FlexAbsOffset{} // reset

	var offset int32
	if err := n.DecodeStrict(&offset); err == nil {
		f.Offset = &offset
		f.Inline = true
		return nil
	}

	var sym string
	if err := n.DecodeStrict(&sym); err == nil {
		f.Sym = &sym
		f.Inline = true
		return nil
	}

	type FlexAbsOffsetData FlexAbsOffset // this works because the MarshalYAML won't be inherited, so it won't result in an infinite loop, but struct tags will remain
	var obj FlexAbsOffsetData
	if err := n.DecodeStrict(&obj); err != nil {
		return fmt.Errorf("line %d: %w", n.Line, err)
	}
	*f = FlexAbsOffset(obj)
	return nil
}

func (f FlexAbsOffset) MarshalYAML() (interface{}, error) {
	if err := f.validate(); err != nil {
		return nil, err
	}
	if f.Inline {
		if f.Offset != nil {
			return f.Offset, nil
		}
		if f.Sym != nil {
			return f.Sym, nil
		}
	}
	type FlexAbsOffsetData FlexAbsOffset // this works because the MarshalYAML won't be inherited, so it won't result in an infinite loop, but struct tags will remain
	return FlexAbsOffsetData(f), nil
}

func (f FlexAbsOffset) Resolve(p *patchlib.Patcher) (int32, error) {
	if err := f.validate(); err != nil {
		return 0, err
	}
	var rel int32
	if f.Rel != nil {
		rel = *f.Rel
	}
	off, err := func() (int32, error) {
		switch {
		case f.Offset != nil:
			return *f.Offset + rel, nil
		case f.Sym != nil:
			return p.ResolveSym(*f.Sym)
		case f.SymPLT != nil:
			return p.ResolveSymPLT(*f.SymPLT)
		case f.SymPLTTail != nil:
			return p.ResolveSymPLTTail(*f.SymPLTTail)
		default:
			panic("this should have been caught by FlexAbsOffset.validate")
		}
	}()
	return off + rel, err
}

func (f FlexAbsOffset) validate() error {
	if f.Offset != nil && *f.Offset < 0 {
		return fmt.Errorf("offset must be positive, got %d", *f.Offset)
	}
	var c int
	for _, v := range []bool{f.Offset != nil, f.Sym != nil, f.SymPLT != nil, f.SymPLTTail != nil} {
		if v {
			c++
		}
	}
	if c == 0 {
		return fmt.Errorf("no offset method specified (%#v)", f)
	}
	if c > 1 {
		return fmt.Errorf("multiple offset methods specified (%#v)", f)
	}
	return nil
}

type Enabled bool
type Description string
type PatchGroup string

type PatchableInstruction interface {
	ApplyTo(*patchlib.Patcher, func(string, ...interface{})) error
}

type BaseAddress FlexAbsOffset
type FindBaseAddressHex string
type FindBaseAddressString string
type FindZlib string
type FindZlibHash string

func (b BaseAddress) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("BaseAddress(%#v)", b)
	offset, err := FlexAbsOffset(b).Resolve(pt)
	if err != nil {
		return fmt.Errorf("BaseAddress: resolve address (%#v): %w", b, err)
	}
	log("  BaseAddress(%#v)", offset)
	return pt.BaseAddress(int32(offset))
}

func (b *BaseAddress) UnmarshalYAML(n *yaml.Node) error {
	return (*FlexAbsOffset)(b).UnmarshalYAML(n)
}

func (b BaseAddress) MarshalYAML() (interface{}, error) {
	return (FlexAbsOffset)(b).MarshalYAML()
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
	Base    *FlexAbsOffset `yaml:"Base,omitempty,flow"` // if specified, Offset is based on this rather than the current offset
	Offset  int32          `yaml:"Offset,omitempty"`
	Find    []byte         `yaml:"Find,omitempty"`
	Replace []byte         `yaml:"Replace,omitempty"`
	// generators
	FindH          *string        `yaml:"FindH,omitempty"`
	ReplaceH       *string        `yaml:"ReplaceH,omitempty"`
	FindInstBLX    *FlexAbsOffset `yaml:"FindInstBLX,omitempty,flow"`
	ReplaceInstBLX *FlexAbsOffset `yaml:"ReplaceInstBLX,omitempty,flow"`
	FindInstBW     *FlexAbsOffset `yaml:"FindInstBW,omitempty,flow"`
	ReplaceInstBW  *FlexAbsOffset `yaml:"ReplaceInstBW,omitempty,flow"`
	ReplaceInstNOP *bool          `yaml:"ReplaceInstNOP,omitempty,flow"` // if specified, must be true
	FindBLX        *uint32        `yaml:"FindBLX,omitempty"`             // Deprecated: Use FindInstBLX instead.
	// special
	CheckOnly *bool `yaml:"CheckOnly,omitempty"` // if specified and true, it will only ensure the presence of the find string
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

func (r ReplaceBytes) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) (perr error) {
	log("ReplaceBytes(%#v)", r)

	// TODO: add tests for new expansions

	curOrig := pt.GetCur()
	cur := curOrig
	if r.Base != nil {
		log("  Base.Resolve(%#v)", *r.Base)
		off, err := r.Base.Resolve(pt)
		if err != nil {
			err = fmt.Errorf("ReplaceBytes: expand Base=%#v: %v", *r.Base, err)
			log("    -> Error: %v", err)
			return err
		}
		cur = off
		log("    -> Offset: 0x%X", off)

		log("    BaseAddress(0x%X) [temp]", cur)
		if err := pt.BaseAddress(cur); err != nil {
			err = fmt.Errorf("ReplaceBytes: update cur to 0x%X: %v", cur, err)
			log("    -> Error: %v", err)
			return err
		}

		defer func() {
			log("  BaseAddress(0x%X) [restore]", curOrig)
			if err := pt.BaseAddress(curOrig); err != nil {
				err = fmt.Errorf("ReplaceBytes: restore overridden cur to 0x%X", curOrig)
				log("  -> Error: %v", err)
				if perr == nil {
					perr = err
				}
				return
			}
		}()
	}

	if r.FindH != nil {
		log("FindH.Expand(%#v)", *r.FindH)
		buf, err := hex.DecodeString(strings.ReplaceAll(*r.FindH, " ", ""))
		if err != nil {
			err = fmt.Errorf("ReplaceBytes: expand FindH=%#v: %v", *r.FindH, err)
			log("  -> Error: %v", err)
			return err
		}
		r.Find = buf
		log("  -> Find = %#v", buf)
	}

	if r.ReplaceH != nil {
		log("ReplaceH.Expand(%#v)", *r.ReplaceH)
		buf, err := hex.DecodeString(strings.ReplaceAll(*r.ReplaceH, " ", ""))
		if err != nil {
			err = fmt.Errorf("ReplaceBytes: expand ReplaceH=%#v: %v", *r.ReplaceH, err)
			log("  -> Error: %v", err)
			return err
		}
		r.Replace = buf
		log("  -> Replace = %#v", buf)
	}

	if r.FindBLX != nil {
		log("FindBLX(0x%X)", *r.FindBLX)
		o := int32(*r.FindBLX)
		r.FindInstBLX = &FlexAbsOffset{Offset: &o}
		log("  -> FindInstBLX = FindBLX = 0x%X", *r.FindBLX)
	}

	if r.FindInstBLX != nil {
		log("FindInstBLX.Expand(%#v)", *r.FindInstBLX)

		log("  FindInstBLX.Resolve(%#v)", *r.FindInstBLX)
		tgt, err := r.FindInstBLX.Resolve(pt)
		if err != nil {
			err = fmt.Errorf("ReplaceBytes: expand FindInstBLX=%#v: %v", *r.FindInstBLX, err)
			log("    -> Error: %v", err)
			return err
		}
		log("    -> Target: 0x%X", tgt)

		pc := cur + r.Offset
		log("  AsmBLX(0x%X, 0x%X)", pc, tgt)
		buf := patchlib.AsmBLX(uint32(pc), uint32(tgt))
		r.Find = buf
		log("    -> Find = %#v", buf)
	}

	if r.ReplaceInstBLX != nil {
		log("ReplaceInstBLX.Expand(%#v)", *r.ReplaceInstBLX)

		log("  ReplaceInstBLX.Resolve(%#v)", *r.ReplaceInstBLX)
		tgt, err := r.ReplaceInstBLX.Resolve(pt)
		if err != nil {
			err = fmt.Errorf("ReplaceBytes: expand ReplaceInstBLX=%#v: %v", *r.ReplaceInstBLX, err)
			log("    -> Error: %v", err)
			return err
		}
		log("    -> Target: 0x%X", tgt)

		pc := cur + r.Offset
		log("  AsmBLX(0x%X, 0x%X)", pc, tgt)
		buf := patchlib.AsmBLX(uint32(pc), uint32(tgt))
		r.Replace = buf
		log("    -> Replace = %#v", buf)
	}

	if r.FindInstBW != nil {
		log("FindInstBW.Expand(%#v)", *r.FindInstBW)

		log("  FindInstBW.Resolve(%#v)", *r.FindInstBW)
		tgt, err := r.FindInstBW.Resolve(pt)
		if err != nil {
			err = fmt.Errorf("ReplaceBytes: expand FindInstB=%#v: %v", *r.FindInstBW, err)
			log("    -> Error: %v", err)
			return err
		}
		log("    -> Target: 0x%X", tgt)

		pc := cur + r.Offset
		log("  AsmBW(0x%X, 0x%X)", pc, tgt)
		buf := patchlib.AsmBW(uint32(pc), uint32(tgt))
		r.Find = buf
		log("    -> Find = %#v", buf)
	}

	if r.ReplaceInstBW != nil {
		log("ReplaceInstBW.Expand(%#v)", *r.ReplaceInstBW)

		log("  ReplaceInstBW.Resolve(%#v)", *r.ReplaceInstBW)
		tgt, err := r.ReplaceInstBW.Resolve(pt)
		if err != nil {
			err = fmt.Errorf("ReplaceBytes: expand ReplaceInstB=%#v: %v", *r.ReplaceInstBW, err)
			log("    -> Error: %v", err)
			return err
		}
		log("    -> Target: 0x%X", tgt)

		pc := cur + r.Offset
		log("  AsmBW(0x%X, 0x%X)", pc, tgt)
		buf := patchlib.AsmBW(uint32(pc), uint32(tgt))
		r.Replace = buf
		log("    -> Replace = %#v", buf)
	}

	if r.ReplaceInstNOP != nil {
		if !*r.ReplaceInstNOP {
			return fmt.Errorf("ReplaceBytes: ReplaceInstNOP must either be true or unspecified")
		}
		// note: must be after all Find expansions, as it depends on checking the length
		log("ReplaceInstNOP.Expand(%#v)", *r.ReplaceInstNOP)
		if len(r.Find)%2 != 0 {
			return fmt.Errorf("ReplaceBytes: find not a multiple of 2 (len=%d)", len(r.Find))
		}
		buf := make([]byte, len(r.Find))
		for i := 0; i < len(buf); i += 2 {
			buf[i], buf[i+1] = 0x00, 0x46
		}
		r.Replace = buf
		log("  -> Replace = %#v", buf)
	}

	if pt.GetCur() != cur {
		panic("why is cur different?")
	}

	if r.CheckOnly != nil && *r.CheckOnly {
		if len(r.Replace) != 0 {
			return fmt.Errorf("ReplaceBytes: CheckOnly is true, but Replace is not empty")
		}
		log("CheckBytes(%#v, %#v)", r.Offset, r.Find)
		log("  ReplaceBytes(%#v, %#v, %#v) [cur:0x%X + off:%d -> abs:0x%X]", r.Offset, r.Find, r.Find, cur, r.Offset, r.Offset+cur)
		return pt.ReplaceBytes(r.Offset, r.Find, r.Find)
	}

	if r.FindInstBLX != nil && r.ReplaceInstBLX != nil {
		if (r.FindInstBLX.SymPLT != nil) != (r.ReplaceInstBLX.SymPLT != nil) {
			return fmt.Errorf("ReplaceBytes: for safety, you cannot replace a BLX to a PLT entry with one to a non-PLT entry or vice-versa (to suppress this warning, split the ReplaceBytes into two steps using placeholder bytes)")
		}
	}

	if r.FindInstBW != nil && r.ReplaceInstBW != nil {
		if (r.FindInstBW.SymPLTTail != nil) != (r.ReplaceInstBW.SymPLTTail != nil) {
			return fmt.Errorf("ReplaceBytes: for safety, you cannot replace a B.W to a PLT entry's tail stub with one to a non-PLT entry's tail stub or vice-versa (to suppress this warning, split the ReplaceBytes into two steps using placeholder bytes)")
		}
	}

	log("ReplaceBytes(%#v, %#v, %#v) [cur:0x%X + off:%d -> abs:0x%X]", r.Offset, r.Find, r.Replace, cur, r.Offset, r.Offset+cur)
	return pt.ReplaceBytes(r.Offset, r.Find, r.Replace)
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

func expandHex(in *string, out *[]byte) (bool, error) {
	if in == nil {
		return false, nil
	}
	buf, err := hex.DecodeString(strings.ReplaceAll(*in, " ", ""))
	if err != nil {
		return true, fmt.Errorf("error expanding shorthand hex `%s`: %w", *in, err)
	}
	*out = buf
	return true, nil
}

// Deprecated
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

// Deprecated: Use BaseAddress instead.
type FindBaseAddressSymbol string

// Deprecated
func (b FindBaseAddressSymbol) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("FindBaseAddressSymbol(%#v)", b)
	return pt.FindBaseAddressSymbol(string(b))
}

// Deprecated: Use Base on ReplaceBytes instead.
type ReplaceBytesAtSymbol struct {
	Symbol   string  `yaml:"Symbol,omitempty"`
	Offset   int32   `yaml:"Offset,omitempty"`
	FindH    *string `yaml:"FindH,omitempty"`
	ReplaceH *string `yaml:"ReplaceH,omitempty"`
	FindBLX  *uint32 `yaml:"FindBLX,omitempty"`
	Find     []byte  `yaml:"Find,omitempty"`
	Replace  []byte  `yaml:"Replace,omitempty"`
}

// Deprecated: Use ReplaceInstNOP on ReplaceBytes instead.
type ReplaceBytesNOP struct {
	Offset  int32   `yaml:"Offset,omitempty"`
	FindH   *string `yaml:"FindH,omitempty"`
	FindBLX *uint32 `yaml:"FindBLX,omitempty"`
	Find    []byte  `yaml:"Find,omitempty"`
}

// Deprecated: Use FindInstBLX and ReplaceInstBLX on ReplaceBytes instead.
type ReplaceBLX struct {
	Offset  int32  `yaml:"Offset,omitempty"`
	Find    uint32 `yaml:"Find"`
	Replace uint32 `yaml:"Replace"`
}

// Deprecated
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

// Deprecated
func (r ReplaceBLX) ApplyTo(pt *patchlib.Patcher, log func(string, ...interface{})) error {
	log("ReplaceBLX(%#v, %#v, %#v)", r.Offset, r.Find, r.Replace)
	return pt.ReplaceBLX(r.Offset, r.Find, r.Replace)
}
