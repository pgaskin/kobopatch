package patchlib

import (
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ianlancetaylor/demangle"
	"rsc.io/arm/armasm"
)

// manually tested with libnickel from 12777, 14622, and 5761 and libadobe from 14622
// should work on any ARM binary targeting the ARMv6 ABI or newer which uses the recommended PLT format (e.g. GCC or Clang)

type dynsym struct {
	// decoded from the dynamic symbol table
	Name   string
	Offset uint32
	Index  uint32
	Type   elf.SymType
	// decoded from the R_ARM_JUMP_SLOT relocs
	OffsetGOT uint32 // optional
	// decoded from the PLT
	OffsetPLT     uint32 // optional
	OffsetPLTTail uint32 // optional
	// generated
	Demangled string // optional
}

func decdynsym(e *elf.File, skipPLTGOT bool) ([]*dynsym, error) {
	if e.Class != elf.ELFCLASS32 && e.Machine != elf.EM_ARM {
		return nil, fmt.Errorf("not a 32-bit arm elf")
	}

	var dynsyms []*dynsym

	// include all dynamic symbols (including the ones without PLT entries)
	edynsyms, err := e.DynamicSymbols()
	if err != nil {
		return nil, fmt.Errorf("get dynamic symbols: %w", err)
	}
	for i, edynsym := range edynsyms {
		if edynsym.Name == "" {
			// discard unnamed symbols (usually just _init, etc stuff)
			continue
		}
		v, err := demangle.ToString(edynsym.Name)
		if err != nil {
			v = ""
		}
		dynsyms = append(dynsyms, &dynsym{
			Name:      edynsym.Name,
			Offset:    uint32(edynsym.Value) &^ 1, // https://static.docs.arm.com/ihi0044/g/aaelf32.pdf: For the purposes of relocation the value used shall be the address of the instruction (st_value &~1).
			Index:     uint32(i + 1),              // Go's DynamicSymbols() preserves the order (thus making the indexes match), but removes the first (null) dynsyn,
			Type:      elf.ST_TYPE(edynsym.Info),
			Demangled: v,
		})
	}

	if skipPLTGOT {
		return dynsyms, nil
	}

	// for each of the dynamic symbols with R_ARM_JUMP_SLOT relocations, add the
	// info from the decoded PLT
	pltrels, err := decpltrel(e)
	if err != nil {
		return nil, fmt.Errorf("read plt relocs: %w", err)
	}
	pltrelidx := map[uint32]elf.Rel32{} // map the symbol index to the reloc for faster access
	for _, pltrel := range pltrels {
		pltrelidx[elf.R_SYM32(pltrel.Info)] = pltrel
	}
	pltents, err := decplt(e)
	if err != nil {
		return nil, fmt.Errorf("decode plt: %w", err)
	}
	pltentidx := map[uint32]pltent{} // map the GOT offset to the PLT entry for faster access
	for _, pltent := range pltents {
		pltentidx[pltent.GOTOffset] = pltent
	}
	for _, s := range dynsyms {
		if s.Type != elf.STT_FUNC {
			// only functions will be in the PLT/GOT
			continue
		}

		// find the GOT offset
		pltrel, ok := pltrelidx[s.Index]
		if !ok {
			// the dynamic symbol may not have a PLT entry
			continue
		}
		if elf.R_ARM(elf.R_TYPE32(pltrel.Info)) != elf.R_ARM_JUMP_SLOT {
			// https://static.docs.arm.com/ihi0044/g/aaelf32.pdf: R_ARM_JUMP_SLOT is
			// used to mark code targets that will be executed. On platforms that
			// support dynamic binding the relocations may be performed lazily on
			// demand. The unresolved address stored in the place will initially
			// point to the entry sequence stub for the dynamic linker and must be
			// adjusted during initial loading by the offset of the load address of
			// the segment from its link address. Addresses stored in the place of
			// these relocations may not be used for pointer comparison until the
			// relocation has been resolved. In a REL form of this relocation the
			// addend, A, is always 0. single entry in the GOT.
			//
			// non-jump slot entries won't be in the decoded PLT/GOT mapping
			continue
		}
		s.OffsetGOT = pltrel.Off

		// find the PLT entry referencing the GOT entry.
		pltent, ok := pltentidx[pltrel.Off]
		if !ok {
			// the GOT entry may not have a PLT one referencing it (this shouldn't happen, but it's not an error if it does)
			continue
		}
		s.OffsetPLT = pltent.PLTOffset
		s.OffsetPLTTail = pltent.PLTOffsetTail
	}
	return dynsyms, nil
}

func decpltrel(e *elf.File) ([]elf.Rel32, error) {
	if e.Class != elf.ELFCLASS32 && e.Machine != elf.EM_ARM {
		return nil, fmt.Errorf("not a 32-bit arm elf")
	}
	relplt := e.Section(".rel.plt")
	if relplt == nil {
		return nil, fmt.Errorf("read .rel.plt: no such section")
	}
	r := relplt.Open()
	var pltrels []elf.Rel32
	for {
		var rel elf.Rel32
		err := binary.Read(r, e.ByteOrder, &rel)
		pltrels = append(pltrels, rel)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read .rel.plt: %w", err)
		}
	}
	return pltrels, nil
}

type pltent struct {
	PLTOffset     uint32
	PLTOffsetTail uint32
	GOTOffset     uint32
}

func decplt(e *elf.File) ([]pltent, error) {
	if e.Class != elf.ELFCLASS32 && e.Machine != elf.EM_ARM {
		return nil, fmt.Errorf("not a 32-bit arm elf")
	}
	// https://stackoverflow.com/a/32808179
	plt := e.Section(".plt")
	if plt == nil {
		return nil, fmt.Errorf("read .plt: no such section")
	}
	buf, err := plt.Data()
	if err != nil {
		return nil, fmt.Errorf("read .plt: %w", err)
	}
	got := e.Section(".got")
	if got == nil {
		return nil, fmt.Errorf("read .got: no such section")
	}
	var asmbufo []uint32
	var asmbufi []armasm.Inst
	var asmbuftail uint32 // non-zero offset of tail call in PLT if it exists for the current entry in asmbuf*
	var pltents []pltent
	pc := uint32(plt.Offset)
	for len(buf) != 0 {
		if len(asmbufi) != len(asmbufo) {
			panic("len(asmbufi) != len(asmbufo)")
		}

		// read the next inst
		t, err := armasm.Decode(buf, armasm.ModeARM)
		if err != nil {
			if len(buf) >= 4 && buf[0] == 0x78 && buf[1] == 0x47 && buf[2] == 0xC0 && buf[3] == 0x46 {
				// Thumb: bx pc    mov r8, r8
				asmbuftail = pc
				buf = buf[4:]
				pc += 4
				continue
			}
			// probably a different thumb instruction, so skip it (note: even if
			// there happens to be another thumb inst (from part of a tail call
			// stub or something similar) which is valid arm, it won't cause
			// issues due to the error checking later)
			buf = buf[2:]
			pc += 2 // some thumb insts are 4 long, but that's fine
			continue
		}
		asmbufo, asmbufi = append(asmbufo, pc), append(asmbufi, t)
		buf = buf[t.Len:]
		pc += uint32(t.Len)

		// we only do the processing at each LDR
		if t.Op != armasm.LDR {
			// if there's more than 8 instructions (just an arbitrary number) in
			// the buffer, something's very wrong (there really should be one at
			// most every 3rd instruction)
			if len(asmbufi) > 8 {
				return nil, fmt.Errorf("parse .plt: at 0x%X: expected LDR instruction somewhere, cur %+q", pc, asmbufi)
			}
			// decode the next inst
			continue
		}

		// each PLT entry should look like 2 ADD instructions then a LDR
		// (technically, it could be different, but this is what the ARM arch
		// guide suggests, and everything I've seen follows it; we also don't
		// want to have to implement a full emulator)
		// https://static.docs.arm.com/ihi0044/g/aaelf32.pdf
		if len(asmbufi) != 3 || asmbufi[0].Op != armasm.ADD || asmbufi[1].Op != armasm.ADD || asmbufi[2].Op != armasm.LDR {
			// discard the junk at the start of the PLT
			if len(pltents) == 0 {
				// if we're more than 32 bytes (just an arbitrary number) into
				// the plt, we have more junk than expected
				if pc-uint32(plt.Offset) > 128 {
					return nil, fmt.Errorf("parse .plt: at 0x%X: more than 128 bytes of junk at start of PLT, cur %+q", pc, asmbufi)
				}
				// reset the buffer
				asmbufo, asmbufi, asmbuftail = nil, nil, 0
				continue
			}
			return nil, fmt.Errorf("parse .plt: at 0x%X: expected 2 ADD instructions before each LDR, got %+q", pc, asmbufi)
		}

		// calculate the got offset by doing a basic emulation of the insts
		reg := map[armasm.Reg]uint32{
			armasm.PC:  0,
			armasm.R12: 0,
			// other registers shouldn't be used
		}
		for n, inst := range asmbufi {
			instpc := asmbufo[n]
			reg[armasm.PC] = instpc + 8 // https://static.docs.arm.com/ddi0406/c/DDI0406C_C_arm_architecture_reference_manual.pdf: In ARM state, the value of the PC is the address of the current instruction plus 8 bytes
			switch inst.Op {
			case armasm.ADD:
				var dst armasm.Reg
				var new uint32
				for i, arg := range inst.Args {
					switch i {
					case 0:
						if arg != armasm.R12 {
							return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: expected dest register for add to be ip (r12)", pc, asmbufi, inst, instpc, i+1)
						}
						dst = armasm.R12
					case 1, 2:
						switch v := arg.(type) {
						case armasm.Reg:
							if rv, ok := reg[v]; !ok {
								return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: unsupported register %s", pc, asmbufi, inst, instpc, i+1, v)
							} else {
								new += rv
							}
						case armasm.Imm:
							new += uint32(v)
						case armasm.ImmAlt:
							new += uint32(v.Imm())
						default:
							return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: unsupported arg type for %#v", pc, asmbufi, inst, instpc, i+1, v)
						}
					case 3:
						if arg != nil {
							return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: expected nothing, got %#v", pc, asmbufi, inst, instpc, i+1, arg)
						}
					default:
						panic("armasm should have returned 4 args...")
					}
				}
				reg[dst] = new
			case armasm.LDR:
				var dst armasm.Reg
				var base armasm.Reg
				var sub bool
				var new uint32
				var fn func()
				for i, arg := range inst.Args {
					switch i {
					case 0:
						if arg != armasm.PC {
							return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: expected dest register for ldr to be pc", pc, asmbufi, inst, instpc, i+1)
						}
						dst = armasm.PC
					case 1:
						switch v := arg.(type) {
						case armasm.Mem:
							if _, ok := reg[v.Base]; !ok {
								return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: unsupported base register %s", pc, asmbufi, inst, instpc, i+1, v.Base)
							} else {
								base = v.Base
							}
							switch v.Mode {
							case armasm.AddrPostIndex:
								fn = func() {
									reg[dst] = reg[base]
									if sub {
										reg[base] = reg[base] - new
									} else {
										reg[base] = reg[base] + new
									}
								}
							case armasm.AddrPreIndex:
								fn = func() {
									if sub {
										reg[dst] = reg[base] - new
										reg[base] = reg[base] - new
									} else {
										reg[dst] = reg[base] + new
										reg[base] = reg[base] + new
									}
								}
							case armasm.AddrOffset:
								fn = func() {
									if sub {
										reg[dst] = reg[base] - new
									} else {
										reg[dst] = reg[base] + new
									}
								}
							default:
								return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: unsupported addressing mode for ldr arg", pc, asmbufi, inst, instpc, i+1)
							}
							if v.Sign != 0 {
								sub = false
								if v.Sign < 0 {
									sub = true
								}
								if rv, ok := reg[v.Index]; !ok {
									return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: unsupported index register %s", pc, asmbufi, inst, instpc, i+1, v.Index)
								} else {
									new = rv
								}
								switch v.Shift {
								case armasm.ShiftLeft:
									new = new << v.Count
								case armasm.ShiftRight:
									new = new >> v.Count
								default:
									return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: unsupported shift mode %s for ldr arg", pc, asmbufi, inst, instpc, i+1, v.Shift)
								}
							} else {
								if v.Offset < 0 {
									sub = true
									new = uint32(v.Offset * -1)
								} else {
									sub = false
									new = uint32(v.Offset)
								}
							}
						default:
							return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: unsupported arg type for %#v", pc, asmbufi, inst, instpc, i+1, v)
						}
					case 2, 3:
						if arg != nil {
							return nil, fmt.Errorf("parse .plt: at 0x%X: parse entry %+q: emulate inst %s at 0x%X: arg %d: expected nothing, got %#v", pc, asmbufi, inst, instpc, i+1, arg)
						}
					default:
						panic("armasm should have returned 4 args...")
					}
				}
				fn()
			default:
				panic("the opcode should have already been checked...")
			}
		}

		if reg[armasm.R12] < uint32(got.Offset) {
			return nil, fmt.Errorf("parse .plt: entry at 0x%X: emulated GOT offset (ip/r12) 0x%X before GOT at 0x%X - 0x%X (size: 0x%X)", asmbufo[0], reg[armasm.R12], got.Offset, got.Offset+got.Size, got.Size)
		}

		pltents = append(pltents, pltent{
			PLTOffset:     asmbufo[0],
			PLTOffsetTail: asmbuftail, // will be 0 if doesn't exist
			GOTOffset:     reg[armasm.R12],
		})

		// reset the buffer for the next entry
		asmbufo, asmbufi, asmbuftail = nil, nil, 0
	}
	seen := map[uint32]int{}
	for i, pltent := range pltents {
		if j, ok := seen[pltent.GOTOffset]; ok {
			return nil, fmt.Errorf("parse .plt: internal error: duplicate emulated got offsets (this is likely a bug) in entry %#v (prev: %#v)", pltent, pltents[j])
		}
		seen[pltent.GOTOffset] = i
	}
	return pltents, nil
}
