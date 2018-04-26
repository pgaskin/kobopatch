package patchlib

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"reflect"
	"runtime/debug"
	"testing"
)

func TestGetBytes(t *testing.T) {
	p := NewPatcher([]byte(`this is a test`))
	eq(t, p.GetBytes(), []byte(`this is a test`), "unexpected output")
}

func TestResetBaseAddress(t *testing.T) {
	p := NewPatcher([]byte(`this is a test`))
	p.cur = 5
	p.ResetBaseAddress()
	eq(t, p.cur, int64(0), "unexpected base address")
}

func TestBaseAddress(t *testing.T) {
	p := NewPatcher([]byte(`this is a test`))
	err(t, p.BaseAddress(14)) // past buf len
	err(t, p.BaseAddress(-1)) // negative
	nerr(t, p.BaseAddress(4))
	eq(t, p.cur, int64(4), "unexpected base address")
}

func TestFindBaseAddress(t *testing.T) {
	p := NewPatcher([]byte(`this is a test`))
	err(t, p.FindBaseAddress([]byte(`thiss`)))
	err(t, p.FindBaseAddress([]byte(`this is a test sdfsdf`)))
	nerr(t, p.FindBaseAddress([]byte(`a test`)))
	eq(t, p.cur, int64(8), "unexpected base address")
}

func TestFindBaseAddressString(t *testing.T) {
	p := NewPatcher([]byte(`this is a test`))
	err(t, p.FindBaseAddressString(`thiss`))
	err(t, p.FindBaseAddressString(`this is a test sdfsdf`))
	nerr(t, p.FindBaseAddressString(`a test`))
	eq(t, p.cur, int64(8), "unexpected base address")
}

func TestReplaceString(t *testing.T) {
	p := NewPatcher([]byte(`this is a test`))
	err(t, p.ReplaceString(0, `this `, `that`))
	nerr(t, p.ReplaceString(0, `this `, `that `))
	err(t, p.ReplaceString(0, `this `, `that `))
	nerr(t, p.ReplaceString(0, `s`, `5`))
	eq(t, p.GetBytes(), []byte(`that i5 a test`), "unexpected output")
}

func TestReplaceBytes(t *testing.T) {
	p := NewPatcher([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05})
	err(t, p.ReplaceBytes(0, []byte{0x00}, []byte{0x00, 0x01}))
	err(t, p.ReplaceBytes(3, []byte{0x02, 0x03}, []byte{0x03, 0x02}))
	nerr(t, p.ReplaceBytes(0, []byte{0x02, 0x03}, []byte{0x03, 0x02}))
	err(t, p.ReplaceBytes(0, []byte{0x02, 0x03}, []byte{0x03, 0x02}))
	eq(t, p.GetBytes(), []byte{0x00, 0x01, 0x03, 0x02, 0x04, 0x05}, "unexpected output")
}

func TestReplaceInt(t *testing.T) {
	p := NewPatcher([]byte{0x00, 0x01, 0x02, 0xff, 0x04, 0x05})
	err(t, p.ReplaceInt(4, 255, 195))
	nerr(t, p.ReplaceInt(1, 255, 195))
	err(t, p.ReplaceInt(1, 255, 195))
	eq(t, p.GetBytes(), []byte{0x00, 0x01, 0x02, 0xc3, 0x04, 0x05}, "unexpected output")
}

func TestReplaceFloat(t *testing.T) {
	p := NewPatcher([]byte{0x00, 0xcd, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xf0, 0x3f, 0x05})
	nerr(t, p.ReplaceFloat(0, 1.05, 3.25))
	err(t, p.ReplaceFloat(0, 1.05, 3.25))
	eq(t, p.GetBytes(), []byte{0x00, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xa, 0x40, 0x05}, "unexpected output")
}

func TestAll(t *testing.T) {
	in := append(
		[]byte{0x00, 0x01, 0xff, 0xcd, 0xcc, 0xcc, 0xcc, 0xcc, 0xcc, 0xf0, 0x3f, 0x01, 0x02, 0x03, 0x04},
		[]byte(`this is a test`)...,
	)
	eout := append(
		[]byte{0x33, 0x33, 0x00, 0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x10, 0x40, 0x01, 0x02, 0x03, 0x04},
		[]byte(`taht is z test`)...,
	)

	p := NewPatcher(in)
	eq(t, p.cur, int64(0), "base address should be correct")

	nerr(t, p.ReplaceInt(0, 255, 0))

	err(t, p.ReplaceString(0, `this `, `that`)) // length mismatch
	nerr(t, p.ReplaceString(0, `this `, `that `))
	err(t, p.ReplaceString(0, `this `, `that `)) // no match

	nerr(t, p.ReplaceFloat(0, 1.05, 4.05))

	err(t, p.FindBaseAddress([]byte("a testdfgdfg"))) // no match
	eq(t, p.cur, int64(0), "base address should be correct")
	nerr(t, p.FindBaseAddress([]byte("a test")))
	eq(t, p.cur, int64(23), "base address should be correct")

	err(t, p.ReplaceString(0, `is`, `si`)) // before current base address
	nerr(t, p.ReplaceString(0, `a`, `t`))

	p.ResetBaseAddress()
	eq(t, p.cur, int64(0), "base address should be correct")
	nerr(t, p.ReplaceString(0, `that`, `taht`))

	nerr(t, p.FindBaseAddress([]byte("taht")))
	eq(t, p.cur, int64(15), "base address should be correct")
	nerr(t, p.ReplaceString(8, `t`, `z`))

	err(t, p.ReplaceBytes(0, []byte{0x00, 0x01}, []byte{0x33, 0x33}))
	p.ResetBaseAddress()
	err(t, p.ReplaceBytes(3, []byte{0x00, 0x01}, []byte{0x33, 0x33}))
	nerr(t, p.ReplaceBytes(0, []byte{0x00, 0x01}, []byte{0x33, 0x33}))

	eq(t, p.GetBytes(), eout, "unexpected output: "+string(p.GetBytes()))
}

func TestReal(t *testing.T) {
	libNickel, err := ioutil.ReadFile("./testdata/libnickel.so.1.0.0")
	nerr(t, err)
	p := NewPatcher(libNickel)

	cs := func(exp string) {
		sum := fmt.Sprintf("%x", sha256.Sum256(p.GetBytes()))
		eq(t, sum, exp, "unexpected checksum: "+sum)
	}

	cs("6603e718eb01947c7face497dd766e3447dce95dbcbabb7d31f46e9d09fbb1e5")

	// Test with select patches to try and find edge cases.

	// Brightness fine control
	p.ResetBaseAddress()
	p.ReplaceInt(0x95DD02, 1, 2)
	cs("7a5c27729c5a2ac20246ad1c44c789410a4ac344a344dab2432b04d0093f186e")

	// Ignore .otf fonts
	p.ResetBaseAddress()
	p.FindBaseAddressString(`*.otf`)
	p.ReplaceString(0, `*`, `_`)
	cs("ed3de9e305b883642d4ecc1e5ace77c15e2d4f6ceacafa1816727724429e34cb")

	// Clock display duration
	p.ResetBaseAddress()
	p.ReplaceBytes(0x9F6252, []byte{0x40, 0xF6, 0xB8, 0x31}, []byte{0x03, 0x21, 0x89, 0x02})
	p.ReplaceInt(0x9F6252, 3, 5)
	cs("61c9d966ef53018c9de9c674dc2946cbb42701fa5a458704bc115023e193c55f")

	// Allow searches on Extra dictionaries
	p.ResetBaseAddress()
	p.FindBaseAddressString("\x00Extra:\x20")
	p.ReplaceString(0007, "\x20", "_")
	cs("8f92a0c8b7041b89d331d5dd0e4579e30caa757272bbc0f47578765f49fe4076")

	// Change dicthtml strings to micthtml
	p.ResetBaseAddress()
	p.ReplaceString(0xC9717C, `%1/dicthtml%2`, `%1/micthtml%2`)
	p.ReplaceString(0xC9718C, `dicthtml`, `micthtml`)
	p.ReplaceString(0xC971D4, `/mnt/onboard/.kobo/dict/dicthtml%1`, `/mnt/onboard/.kobo/dict/micthtml%1`)
	cs("8a36efb87ccfd3fe0bad3a9913be12273fec089cbb7537e6a4cff69edcae1520")

	// Un-force link decoration in KePubs
	p.ResetBaseAddress()
	p.FindBaseAddressString(`a:link, a:visited, a:hover, a:active {`)
	p.ReplaceString(0x0027, "b", "_")
	p.ReplaceString(0x0053, "c", "_")
	cs("a64e265813aaf58d5fb227d681a9049ebd2daf1270a1cb3f9d21564d1f260842")

	// KePub stylesheet additions
	p.ResetBaseAddress()
	p.FindBaseAddressString(".KBHighlighting, .KBSearchResult {")
	p.ReplaceString(0x0000, ".KBHighlighting, .KBSearchResult { background-color: #C6C6C6 !important; } \t", ".KBHighlighting,.KBSearchResult{background-color:#C6C6C6!important}.KBSearch")
	p.ReplaceString(0x004C, ".KBSearchResult, .KBAnnotation, .KBHighlighting { font-size: 100% !important; -webkit-text-combine: inherit !important; } \t", "Result,.KBAnnotation,.KBHighlighting{font-size:100%!important;-webkit-text-combine:inherit!important}.KBAnnotation[writingM")
	p.ReplaceString(0x00C7, ".KBAnnotation[writingMode=\"horizontal-tb\"], .KBAnnotationContinued[writingMode=\"horizontal-tb\"] { border-bottom: 2px solid black !important; } \t", "ode=\"horizontal-tb\"],.KBAnnotationContinued[writingMode=\"horizontal-tb\"]{border-bottom:2px solid black!important}.KBAnnotation[writingMode=\"vert")
	p.ReplaceString(0x0157, ".KBAnnotation[writingMode=\"vertical-rl\"], .KBAnnotationContinued[writingMode=\"vertical-rl\"] { border-right: 2px solid black !important; } \t", "ical-rl\"],.KBAnnotationContinued[writingMode=\"vertical-rl\"]{border-right:2px solid black!important}.KBAnnotation[writingMode=\"vertical-lr\"]")
	p.ReplaceString(0x01E2, ".KBAnnotation[writingMode=\"vertical-lr\"], .KBAnnotationContinued[writingMode=\"vertical-lr\"] { border-left: 2px solid black !important; }", ",.KBAnnotationContinued[writingMode=\"vertical-lr\"]{border-left:2px solid black!important}/*********************************************/")
	cs("07509cc3ed09e60558b37f4f71245af8893b52f221b84810260be53c0f163f6f")

	// My 10 line spacing values
	p.ResetBaseAddress()
	p.ReplaceBytes(0x659DA4, []byte{0xBE, 0xF5, 0xAE, 0xE9}, []byte{0x00, 0x46, 0x00, 0x46})
	p.ReplaceBytes(0x659DFA, []byte{0xBE, 0xF5, 0x84, 0xE9}, []byte{0x00, 0x46, 0x00, 0x46})
	p.ReplaceBytes(0x659E24, []byte{0xBE, 0xF5, 0x6E, 0xE9}, []byte{0x00, 0x46, 0x00, 0x46})
	p.ReplaceBytes(0x659E60, []byte{0xBE, 0xF5, 0x50, 0xE9}, []byte{0x00, 0x46, 0x00, 0x46})
	p.ReplaceBytes(0x659EC6, []byte{0xBE, 0xF5, 0x1E, 0xE9}, []byte{0x00, 0x46, 0x00, 0x46})
	p.BaseAddress(0x659F60)
	p.ReplaceFloat(0x0000, 1.05, 0.8)
	p.ReplaceFloat(0x0008, 1.07, 0.85)
	p.ReplaceFloat(0x0010, 1.1, 0.875)
	p.ReplaceFloat(0x0018, 1.35, 0.9)
	p.ReplaceFloat(0x0020, 1.7, 0.925)
	p.ReplaceFloat(0x0028, 1.8, 0.95)
	p.ReplaceFloat(0x0030, 2.2, 0.975)
	p.ReplaceFloat(0x0038, 2.4, 1.0)
	p.ReplaceFloat(0x0040, 2.6, 1.05)
	p.ReplaceFloat(0x0048, 2.8, 1.1)
	cs("d07f0d59517bee75043505da790adfe4875b18eea2f7d65cfd6bb7e61068ecc9")
}

func nerr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("err should be nil: %v", err)
		debug.PrintStack()
	}
}

func err(t *testing.T, err error) {
	if err == nil {
		t.Error("err should not be nil")
		debug.PrintStack()
	}
}

func eq(t *testing.T, a, b interface{}, msg string) {
	if !reflect.DeepEqual(a, b) {
		t.Error(msg)
		debug.PrintStack()
	}
}
