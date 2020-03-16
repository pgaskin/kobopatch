package patchlib

import "encoding/binary"

// note: this is the 32-bit thumb-2 instruction encoding, not thumb-1
// see the thumb-2 reference manual, section 4.6.18

// AsmBW assembles a B.W instruction and returns a byte slice which can be patched
// directly into a binary.
func AsmBW(pc, target uint32) []byte {
	return mustBytes(toBEBin(bw(pc, target)))
}

// AsmBLX assembles a BLX instruction and returns a byte slice which can be patched
// directly into a binary.
func AsmBLX(pc, target uint32) []byte {
	return mustBytes(toBEBin(blx(pc, target)))
}

// Thumb-2 reference manual, 4.6.12
// B.W (encoding T4) (no cond) (thumb to thumb)
// 1 1 1 1 0 s imm10 1 0 J1 1 J2 imm11
//
//     I1    = NOT(J1 EOR S)                            thus...  J1    = NOT(I1) EOR S
//     I2    = NOT(J2 EOR S)                            thus...  J2    = NOT(I2) EOR S
//     imm32 = SignExtend(S:I1:I2:imm10:imm11:'0', 32)  thus...  S     = SIGN(imm32)
//                                                               I1    = imm32[24]
//                                                               I2    = imm32[23]
//                                                               imm10 = imm32[12:22]
//                                                               imm11 = imm32[1:12]
//
//     BranchWritePC(PC + imm32)                        thus...  imm32 = target - PC - 4
//
//     imm32 must be between -16777216 and 16777214
func bw(PC, target uint32) uint32 {
	var imm32 int32
	var imm11, imm10, S, I2, I1, J2, J1 uint32
	imm32 = int32(target) - int32(PC) - 4
	imm11 = uint32(imm32>>1) & 0b11111111111 // imm32[1:12]
	imm10 = uint32(imm32>>12) & 0b1111111111 // imm32[12:22]
	I2 = bi((imm32>>22)&1 != 0)              // imm32[22]
	I1 = bi((imm32>>23)&1 != 0)              // imm32[23]
	S = bi(imm32 < 0)                        // SIGN(imm32)
	J2 = (^I2 ^ S) & 1
	J1 = (^I1 ^ S) & 1

	var inst uint32
	inst |= uint32(1) << 31
	inst |= uint32(1) << 30
	inst |= uint32(1) << 29
	inst |= uint32(1) << 28
	inst |= uint32(0) << 27
	inst |= uint32(S) << 26
	inst |= uint32(imm10) << 16 // 17 18 19 20 21 22 23 24 25
	inst |= uint32(1) << 15
	inst |= uint32(0) << 14
	inst |= uint32(J1) << 13
	inst |= uint32(1) << 12
	inst |= uint32(J2) << 11
	inst |= uint32(imm11) << 0 // 1 2 3 4 5 6 7 8 9 10

	lebuf := make([]byte, 4)
	lebuf[0] = uint8(inst >> 8 & 0xFF)
	lebuf[1] = uint8(inst >> 0 & 0xFF)
	lebuf[2] = uint8(inst >> 24 & 0xFF)
	lebuf[3] = uint8(inst >> 16 & 0xFF)

	return binary.LittleEndian.Uint32(lebuf) // le to sys endianness
}

// Thumb-2 reference manual, 4.6.18
// BLX (encoding T2) (no cond) (thumb to arm)
// 1 1 1 1 0 S imm10H 1 1 J1 0 J2 imm10L 0
//
//     I1    = NOT(J1 EOR S)                               thus...  J1     = NOT(I1) EOR S
//     I2    = NOT(J2 EOR S)                               thus...  J2     = NOT(I2) EOR S
//     imm32 = SignExtend(S:I1:I2:imm10H:imm10L:'00', 32)  thus...  S      = SIGN(imm32)
//                                                                  I1     = imm32[24]
//                                                                  I2     = imm32[23]
//                                                                  imm10H = imm32[12:22]
//                                                                  imm10L = imm32[2:12]
//
//     next_instr_addr = PC                             n/a (for the return address)
//     LR              = next_instr_addr<31:1>:'1'      n/a (for the return address)
//     SelectInstrSet(InstrSet_ARM)                     n/a (for the target)
//     BranchWritePC(Align(PC, 4) + imm32)              thus...  imm32 = target - (pc & ~3) - 4
//
//     imm32 must be multiples of 4 between -16777216 and 16777212
func blx(PC, target uint32) uint32 {
	var imm32 int32
	var imm10L, imm10H, S, I2, I1, J2, J1 uint32
	imm32 = int32(target) - int32(PC&^3) - 4
	imm10L = uint32(imm32>>2) & 0b1111111111  // imm32[2:12]
	imm10H = uint32(imm32>>12) & 0b1111111111 // imm32[12:22]
	I2 = bi((imm32>>22)&1 != 0)               // imm32[22]
	I1 = bi((imm32>>23)&1 != 0)               // imm32[23]
	S = bi(imm32 < 0)                         // SIGN(imm32)
	J2 = (^I2 ^ S) & 1
	J1 = (^I1 ^ S) & 1

	var inst uint32
	inst |= uint32(1) << 31
	inst |= uint32(1) << 30
	inst |= uint32(1) << 29
	inst |= uint32(1) << 28
	inst |= uint32(0) << 27
	inst |= uint32(S) << 26
	inst |= uint32(imm10H) << 16 // 17 18 19 20 21 22 23 24 25
	inst |= uint32(1) << 15
	inst |= uint32(1) << 14
	inst |= uint32(J1) << 13
	inst |= uint32(0) << 12
	inst |= uint32(J2) << 11
	inst |= uint32(imm10L) << 1 // 2 3 4 5 6 7 8 9 10
	inst |= uint32(0) << 0

	lebuf := make([]byte, 4)
	lebuf[0] = uint8(inst >> 8 & 0xFF)
	lebuf[1] = uint8(inst >> 0 & 0xFF)
	lebuf[2] = uint8(inst >> 24 & 0xFF)
	lebuf[3] = uint8(inst >> 16 & 0xFF)

	return binary.LittleEndian.Uint32(lebuf) // le to sys endianness
}

func bi(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}
