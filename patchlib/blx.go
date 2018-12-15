package patchlib

// go port of https://gist.github.com/jeremy-allen-cs/c93bd333b5b585c2b840

func blx(pc, target uint32) uint32 {
	pc += 4          // arm pipeline
	pc &= 0xfffffffc // align to 4 bytes
	offset := target - pc

	s := getBits(offset, 24, 24)
	i1 := getBits(offset, 23, 23)
	i2 := getBits(offset, 22, 22)
	imm10h := getBits(offset, 12, 21)
	imm10l := getBits(offset, 2, 11)

	j1, j2 := i1^1, i2^1
	j1, j2 = j1^s, j2^s

	var inst uint32
	inst = setBits(inst, 27, 0x1e, 5)    // bits 27-31 = 0b11110
	inst = setBit(inst, 26, s != 0)      // bit 26 = S
	inst = setBits(inst, 16, imm10h, 10) // imm10h, bits 16-25
	inst = setBits(inst, 14, 0x3, 2)     // static
	inst = setBit(inst, 13, j1 != 0)     // j1
	inst = setBit(inst, 12, false)       // j2
	inst = setBit(inst, 11, j2 != 0)     // j2
	inst = setBits(inst, 1, imm10l, 10)  // imm10l
	inst = setBit(inst, 0, false)        // last bit

	top, bot := 0xffff0000&uint32(inst), 0x0000ffff&uint32(inst)
	top, bot = uint32(swapBytes(uint16(top>>16))), uint32(swapBytes(uint16(bot)))

	return uint32(top)<<16 | uint32(bot)
}

func swapBytes(word uint16) uint16 {
	a, b := word&0x00ff, word&0xff00
	return a<<8 | b>>8
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
		mask = mask + (1 << i)
	}
	return val >> start & mask
}

func setBits(val uint32, offset uint, bits uint32, nbits uint) uint32 {
	for i := uint(0); i < nbits; i++ {
		val = setBit(val, i+offset, getBit(bits, i))
	}
	return val
}
