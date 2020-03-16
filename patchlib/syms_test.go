package patchlib

import (
	"debug/elf"
	"os"
	"testing"
)

func TestSyms(t *testing.T) {
	f, err := os.Open("./testdata/libnickel.so.1.0.0")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	e, err := elf.NewFile(f)
	if err != nil {
		panic(err)
	}
	defer e.Close()

	dynsyms, err := decdynsym(e, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	for i, s := range e.Sections {
		if s.Name == ".plt" {
			e.Sections = append(e.Sections[:i], e.Sections[i+1:]...)
			break
		}
	}
	if _, err := decdynsym(e, true); err != nil {
		t.Errorf("unexpected error decoding elf with corrupt plt with pltgot skipped: %v", err)
	}

	_ = dynsyms
	// TODO: test some random symbols and ensure they are correct
}
