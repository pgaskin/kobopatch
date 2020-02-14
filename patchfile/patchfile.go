// Package patchfile provides a standard interface to read patchsets from files.
package patchfile

import (
	"fmt"
	"io/ioutil"

	"github.com/geek1011/kobopatch/patchlib"
)

// Log is used to log debugging messages.
var Log = func(format string, a ...interface{}) {}

// PatchSet represents a set of patches which can be applied to a Patcher.
type PatchSet interface {
	// Validate validates the PatchSet.
	Validate() error
	// ApplyTo applies a PatchSet to a Patcher.
	ApplyTo(*patchlib.Patcher) error
	// SetEnabled sets the Enabled state of a Patch in a PatchSet.
	SetEnabled(string, bool) error
}

var formats = map[string]func([]byte) (PatchSet, error){}

// RegisterFormat registers a format.
func RegisterFormat(name string, f func([]byte) (PatchSet, error)) {
	if _, ok := formats[name]; ok {
		panic("attempt to register duplicate format " + name)
	}
	formats[name] = f
}

// GetFormat gets a format.
func GetFormat(name string) (func([]byte) (PatchSet, error), bool) {
	f, ok := formats[name]
	return f, ok
}

// GetFormats gets all registered formats.
func GetFormats() []string {
	f := []string{}
	for n := range formats {
		f = append(f, n)
	}
	return f
}

// ReadFromFile reads a patchset from a file (but does not validate it).
func ReadFromFile(format, filename string) (PatchSet, error) {
	f, ok := GetFormat(format)
	if !ok {
		return nil, fmt.Errorf("no format called '%s'", format)
	}

	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open patch file: %w", err)
	}

	ps, err := f(buf)
	if err != nil {
		return nil, fmt.Errorf("could not parse patch file: %w", err)
	}

	return ps, nil
}
