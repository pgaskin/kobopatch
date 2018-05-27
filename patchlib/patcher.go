// Package patchlib provides common functions related to patching binaries.
package patchlib

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Patcher applies patches to a byte array. All operations are done starting from cur.
type Patcher struct {
	buf []byte
	cur int32
}

// NewPatcher creates a new Patcher.
func NewPatcher(in []byte) *Patcher {
	return &Patcher{in, 0}
}

// GetBytes returns the current content of the Patcher.
func (p *Patcher) GetBytes() []byte {
	return p.buf
}

// ResetBaseAddress moves cur to 0.
func (p *Patcher) ResetBaseAddress() {
	p.cur = 0
}

// BaseAddress moves cur to an offset. The offset starts at 0.
func (p *Patcher) BaseAddress(offset int32) error {
	if offset < 0 {
		return errors.New("BaseAddress: offset less than 0")
	}
	if offset >= int32(len(p.buf)) {
		return errors.New("BaseAddress: offset greater than length of buf")
	}
	p.cur = offset
	return nil
}

// FindBaseAddress moves cur to the offset of a sequence of bytes.
func (p *Patcher) FindBaseAddress(find []byte) error {
	if len(find) > len(p.buf) {
		return errors.New("FindBaseAddress: length of bytes to find greater than buf")
	}

	i := bytes.Index(p.buf, find)
	if i < 0 {
		return errors.New("FindBaseAddress: could not find bytes")
	}
	p.cur = int32(i)

	return nil
}

// FindBaseAddressString moves cur to the offset of a string.
func (p *Patcher) FindBaseAddressString(find string) error {
	return p.FindBaseAddress([]byte(find))
}

// ReplaceBytes replaces the first occurrence of a sequence of bytes with another of the same length.
func (p *Patcher) ReplaceBytes(offset int32, find, replace []byte) error {
	return wrapErrIfNotNil("ReplaceBytes", p.replaceValue(offset, find, replace))
}

// ReplaceString replaces the first occurrence of a string with another of the same length.
func (p *Patcher) ReplaceString(offset int32, find, replace string) error {
	if len(replace) < len(find) {
		// If replacement shorter than find, append a null to the replacement string to be consistent with the original patch32lsb.
		replace += "\x00"
		replace = replace + find[len(replace):]
	}
	return wrapErrIfNotNil("ReplaceString", p.replaceValue(offset, find, replace))
}

// ReplaceInt replaces the first occurrence of an integer between 0 and 255 inclusively.
func (p *Patcher) ReplaceInt(offset int32, find, replace uint8) error {
	return wrapErrIfNotNil("ReplaceInt", p.replaceValue(offset, find, replace))
}

// ReplaceFloat replaces the first occurrence of a float.
func (p *Patcher) ReplaceFloat(offset int32, find, replace float64) error {
	return wrapErrIfNotNil("ReplaceFloat", p.replaceValue(offset, find, replace))
}

// replaceValue encodes find and replace as little-endian binary and replaces the first
// occurrence starting at cur. The lengths of the encoded find and replace must be the
// same, or an error will be returned.
func (p *Patcher) replaceValue(offset int32, find, replace interface{}) error {
	if int32(len(p.buf)) < p.cur+offset {
		return errors.New("offset past end of buf")
	}

	var err error
	var fbuf, rbuf []byte

	if fstr, ok := find.(string); ok {
		fbuf = []byte(fstr)
	} else {
		fbuf, err = toLEBin(find)
		if err != nil {
			return fmt.Errorf("could not encode find: %v", err)
		}
	}

	if rstr, ok := replace.(string); ok {
		rbuf = []byte(rstr)
	} else {
		rbuf, err = toLEBin(replace)
		if err != nil {
			return fmt.Errorf("could not encode replace: %v", err)
		}
	}

	if len(fbuf) != len(rbuf) {
		return errors.New("length mismatch in byte replacement")
	}
	if int32(len(p.buf)) < p.cur+offset+int32(len(fbuf)) {
		return errors.New("replaced value past end of buf")
	}

	if !bytes.Contains(p.buf[p.cur+offset:], fbuf) {
		return errors.New("could not find specified bytes")
	}

	copy(p.buf[p.cur+offset:], bytes.Replace(p.buf[p.cur+offset:], fbuf, rbuf, 1))
	return nil
}

func toLEBin(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, v)
	return buf.Bytes(), err
}

func wrapErrIfNotNil(txt string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %v", txt, err)
	}
	return nil
}
