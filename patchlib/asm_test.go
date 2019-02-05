package patchlib

import (
	"bytes"
	"fmt"
	"testing"
)

func TestMovR0Bool(t *testing.T) {
	if !bytes.Equal(MovR0Bool(true), []byte{0x4F, 0xF0, 0x01, 0x00}) {
		t.Errorf("expected true to return 4F F0 01 00 (MOV.W r0, #1)")
	}
	if !bytes.Equal(MovR0Bool(false), []byte{0x4F, 0xF0, 0x00, 0x00}) {
		t.Errorf("expected true to return 4F F0 00 00 (MOV.W r0, #1)")
	}
}

func TestBLX(t *testing.T) {
	for _, tc := range []struct{ pc, target, inst uint32 }{
		{0x83EDE8, 0x40EF40, 0xD0F7AAE0},
		{0x83EDE8, 0x41A4A0, 0xDBF75AE3},
		{0x83D426, 0x40EF40, 0xD1F78CE5},
		{0x83D426, 0x41A4A0, 0xDDF73CE0},
	} {
		t.Run(fmt.Sprintf("%X_%X", tc.pc, tc.target), func(t *testing.T) {
			if inst := blx(tc.pc, tc.target); inst != tc.inst {
				t.Errorf("%X: BLX #0x%X - expected %X, got %X", tc.pc, tc.target, tc.inst, inst)
			}
		})
	}
}

func TestSwapBytes(t *testing.T) {
	for _, tc := range []struct{ a, b uint16 }{
		{0xABCD, 0xCDAB},
		{0x0000, 0x0000},
		{0xFFFF, 0xFFFF},
		{0xFF00, 0x00FF},
	} {
		t.Run(fmt.Sprintf("%X", tc.a), func(t *testing.T) {
			if b := swapBytes(tc.a); b != tc.b {
				t.Errorf("swap %X - expected %X, got %X", tc.a, tc.b, b)
			}
		})
	}
}

func TestGetBit(t *testing.T) {
	for _, tc := range []struct {
		a uint32
		b uint
		c bool
	}{
		{0, 1, false},
		{1, 1, false},
		{0, 3, false},
		{8, 0, false},
		{8, 3, true},
		{1, 0, true},
		{2, 1, true},
	} {
		t.Run(fmt.Sprintf("%b_%d", tc.a, tc.b), func(t *testing.T) {
			if c := getBit(tc.a, tc.b); c != tc.c {
				t.Errorf("get bit %d of %b - expected %t, got %t", tc.b, tc.a, tc.c, c)
			}
		})
	}
}

func TestSetBit(t *testing.T) {
	for _, tc := range []struct {
		a uint32
		b uint
		c bool
		d uint32
	}{
		{0, 0, true, 1},
		{0, 1, true, 2},
		{7, 0, true, 7},
		{7, 0, false, 6},
	} {
		t.Run(fmt.Sprintf("%b_%d_%t", tc.a, tc.b, tc.c), func(t *testing.T) {
			if d := setBit(tc.a, tc.b, tc.c); d != tc.d {
				t.Errorf("set bit %d of %b to %t - expected %b, got %b", tc.b, tc.a, tc.c, tc.d, d)
			}
		})
	}
}

func TestGetBits(t *testing.T) {
	for _, tc := range []struct {
		a    uint32
		b, c uint
		d    uint32
	}{
		{123, 1, 4, 13},
		{123, 2, 4, 6},
		{123, 2, 3, 2},
		{123, 6, 7, 1},
		{123, 2, 2, 0},
		{123, 7, 8, 0},
	} {
		t.Run(fmt.Sprintf("%b_%d_%d", tc.a, tc.b, tc.c), func(t *testing.T) {
			if d := getBits(tc.a, tc.b, tc.c); d != tc.d {
				t.Errorf("get bits %d-%d of %b - expected %b, got %b", tc.b, tc.c, tc.a, tc.d, d)
			}
		})
	}
}

func TestSetBits(t *testing.T) {
	for _, tc := range []struct {
		a uint32
		b uint
		c uint32
		d uint
		e uint32
	}{
		{0, 1, 1, 1, 2},
		{6, 1, 1, 1, 6},
		{7, 1, 1, 3, 3},
		{7, 6, 1, 3, 71},
		{7, 6, 3, 3, 199},
	} {
		t.Run(fmt.Sprintf("%b_%d_%b_%d", tc.a, tc.b, tc.c, tc.d), func(t *testing.T) {
			if e := setBits(tc.a, tc.b, tc.c, tc.d); e != tc.e {
				t.Errorf("set %d bits of %b on %b at offset %d - expected %b, got %b", tc.d, tc.c, tc.a, tc.b, tc.e, e)
			}
		})
	}
}
