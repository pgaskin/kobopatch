package patchlib

import (
	"fmt"
	"testing"
)

// Note: kstool thumb "B..." 0xOFFSET

func TestAsmBW(t *testing.T) {
	for _, tc := range []struct{ pc, target, inst uint32 }{
		{0x83EDE8, 0x40EF40, 0xD0F7AAB0},
		{0x83EDE8, 0x41A4A0, 0xDBF75AB3},
		{0x83D426, 0x40EF40, 0xD1F78BB5},
		{0x83D426, 0x41A4A0, 0xDDF73BB0},
		{0x02EFE6, 0x0189C8, 0xE9F7EFBC},
	} {
		t.Run(fmt.Sprintf("%X_%X", tc.pc, tc.target), func(t *testing.T) {
			if inst := bw(tc.pc, tc.target); inst != tc.inst {
				t.Errorf("%X: B.W #0x%X - expected %X, got %X", tc.pc, tc.target, tc.inst, inst)
			} else if fmt.Sprintf("%X", inst) != fmt.Sprintf("%X", AsmBW(tc.pc, tc.target)) {
				t.Errorf("mismatch between []byte and uint32 versions")
			}
		})
	}
}

func TestAsmBLX(t *testing.T) {
	for _, tc := range []struct{ pc, target, inst uint32 }{
		{0x83EDE8, 0x40EF40, 0xD0F7AAE0},
		{0x83EDE8, 0x41A4A0, 0xDBF75AE3},
		{0x83D426, 0x40EF40, 0xD1F78CE5},
		{0x83D426, 0x41A4A0, 0xDDF73CE0},
	} {
		t.Run(fmt.Sprintf("%X_%X", tc.pc, tc.target), func(t *testing.T) {
			if inst := blx(tc.pc, tc.target); inst != tc.inst {
				t.Errorf("%X: BLX #0x%X - expected %X, got %X", tc.pc, tc.target, tc.inst, inst)
			} else if fmt.Sprintf("%X", inst) != fmt.Sprintf("%X", AsmBLX(tc.pc, tc.target)) {
				t.Errorf("mismatch between []byte and uint32 versions")
			}
		})
	}
}
