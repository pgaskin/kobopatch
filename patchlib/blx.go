package patchlib

// note: this is the 32-bit thumb-2 instruction encoding, not thumb-1
// see the thumb-2 reference manual, section 4.6.18

// AsmB assembles a B instruction and returns a byte slice which can be patched
// directly into a binary.
func AsmB(pc, target uint32) []byte {
	return mustBytes(toBEBin(b(pc, target)))
}

// AsmBL assembles a BL instruction and returns a byte slice which can be patched
// directly into a binary.
func AsmBL(pc, target uint32) []byte {
	return mustBytes(toBEBin(bl(pc, target)))
}

// AsmBLX assembles a BLX instruction and returns a byte slice which can be patched
// directly into a binary.
func AsmBLX(pc, target uint32) []byte {
	return mustBytes(toBEBin(blx(pc, target)))
}

func b(pc, target uint32) uint32   { return branch(pc, target, false, true) }
func bl(pc, target uint32) uint32  { return branch(pc, target, true, true) }
func blx(pc, target uint32) uint32 { return branch(pc, target, true, false) }

func branch(pc, target uint32, b14, b12 bool) uint32 {
	pc += 4  // thumb pipeline
	pc &^= 3 // align to 4 bytes
	offset := target - pc

	s := getBits(offset, 24, 24)
	i1 := getBits(offset, 23, 23)
	i2 := getBits(offset, 22, 22)
	imm10h := getBits(offset, 12, 21)
	imm1Xl := getBits(offset, 1, 11)

	j1, j2 := (i1^1)^s, (i2^1)^s

	var inst uint32
	inst = setBits(inst, 27, 0x1e, 5)    // static (branch) (bits 27-31 = 11110)
	inst = setBit(inst, 26, s != 0)      // bit 26 = S
	inst = setBits(inst, 16, imm10h, 10) // imm10h, bits 16-25
	inst = setBit(inst, 15, true)        // static (for b/bl/blx)
	inst = setBit(inst, 14, b14)         // static (differs between b/bl/blx)
	inst = setBit(inst, 13, j1 != 0)     // j1
	inst = setBit(inst, 12, b12)         // static (differs between b/bl/blx)
	inst = setBit(inst, 11, j2 != 0)     // j2
	inst = setBits(inst, 0, imm1Xl, 11)  // imm10l / imm11
	if !b12 {
		inst = setBit(inst, 0, false) // rightmost bit is 0 for blx
	}

	return uint32(swapBytes(uint16((0xffff0000&inst)>>16)))<<16 | uint32(swapBytes(uint16(0x0000ffff&inst)))
}

func swapBytes(word uint16) uint16 {
	return (word&0x00ff)<<8 | (word&0xff00)>>8
}

func getBit(val uint32, idx uint) bool {
	return (val & (1 << idx)) != 0
}

func setBit(val uint32, idx uint, x bool) uint32 {
	mask := uint32(1 << idx)
	val &= ^mask
	if x {
		val |= mask
	}
	return val
}

func getBits(val uint32, start, end uint) uint32 {
	var mask uint32
	for i := uint(0); i <= end-start; i++ {
		mask += (1 << i)
	}
	return val >> start & mask
}

func setBits(val uint32, offset uint, bits uint32, nbits uint) uint32 {
	for i := uint(0); i < nbits; i++ {
		val = setBit(val, i+offset, getBit(bits, i))
	}
	return val
}
