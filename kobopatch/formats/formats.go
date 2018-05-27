package formats

import (
	"io/ioutil"

	"github.com/geek1011/kobopatch/patchlib"
	"github.com/pkg/errors"
)

// Log is used to log debugging messages.
var Log = func(format string, a ...interface{}) {}

// PatchSet represents a set of patches which can be applied to a Patcher.
type PatchSet interface {
	// Validate validates the PatchSet.
	Validate() error
	// ApplyTo applies a PatchSet to a Patcher.
	ApplyTo(*patchlib.Patcher) error
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

// ReadFromFile reads a patchset from a file (but does not validate it).
func ReadFromFile(format, filename string) (PatchSet, error) {
	f, ok := GetFormat(format)
	if !ok {
		return nil, errors.Errorf("no format called '%s'", format)
	}

	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "could not open patch file")
	}

	ps, err := f(buf)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse patch file")
	}

	return ps, nil
}
