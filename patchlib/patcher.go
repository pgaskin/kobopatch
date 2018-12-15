// Package patchlib provides common functions related to patching binaries.
package patchlib

import (
	"bytes"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"strings"
	"unicode/utf8"

	"github.com/DataDog/czlib"
	"github.com/pkg/errors"
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
	return wrapErrIfNotNil("ReplaceBytes", p.replaceValue(offset, find, replace, true))
}

// ReplaceString replaces the first occurrence of a string with another of the same length.
func (p *Patcher) ReplaceString(offset int32, find, replace string) error {
	if len(replace) < len(find) {
		// If replacement shorter than find, append a null to the replacement string to be consistent with the original patch32lsb.
		replace += "\x00"
		replace = replace + find[len(replace):]
	}
	return wrapErrIfNotNil("ReplaceString", p.replaceValue(offset, find, replace, false))
}

// ReplaceInt replaces the first occurrence of an integer between 0 and 255 inclusively.
func (p *Patcher) ReplaceInt(offset int32, find, replace uint8) error {
	return wrapErrIfNotNil("ReplaceInt", p.replaceValue(offset, find, replace, true))
}

// ReplaceFloat replaces the first occurrence of a float.
func (p *Patcher) ReplaceFloat(offset int32, find, replace float64) error {
	return wrapErrIfNotNil("ReplaceFloat", p.replaceValue(offset, find, replace, true))
}

// FindZlib finds the base address of a zlib css stream based on a substring (not sensitive to whitespace).
func (p *Patcher) FindZlib(find string) error {
	if len(find) > len(p.buf) {
		return errors.New("FindZlib: length of string to find greater than buf")
	}
	z, err := p.ExtractZlib()
	if err != nil {
		return errors.Wrap(err, "FindZlib: could not extract zlib streams")
	}
	var i int32
	for _, zi := range z {
		if strings.Contains(zi.CSS, find) || strings.Contains(stripWhitespace(zi.CSS), stripWhitespace(find)) {
			if i != 0 {
				return errors.New("FindZlib: substring to find is not unique")
			}
			i = zi.Offset
			continue
		}
		// Handle minification from below
		zi.CSS = strings.Replace(zi.CSS, "\n    ", "\n", -1)
		zi.CSS = strings.Replace(zi.CSS, "\n  ", "\n", -1)
		zi.CSS = strings.Replace(zi.CSS, "\n ", "\n", -1)
		findm := strings.Replace(find, "\n  ", "\n", -1)
		findm = strings.Replace(find, "\n ", "\n", -1)
		findm = strings.Replace(findm, "\n    ", "\n", -1)
		if strings.Contains(zi.CSS, findm) || strings.Contains(stripWhitespace(zi.CSS), stripWhitespace(findm)) {
			if i != 0 {
				return errors.New("FindZlib: substring to find is not unique")
			}
			i = zi.Offset
			continue
		}
		zi.CSS = strings.Replace(zi.CSS, ": ", ":", -1)
		zi.CSS = strings.Replace(zi.CSS, " {", "{", -1)
		findm = strings.Replace(findm, ": ", ":", -1)
		findm = strings.Replace(findm, " {", "{", -1)
		if strings.Contains(zi.CSS, findm) || strings.Contains(stripWhitespace(zi.CSS), stripWhitespace(findm)) {
			if i != 0 {
				return errors.New("FindZlib: substring to find is not unique")
			}
			i = zi.Offset
			continue
		}
		zi.CSS = strings.Replace(zi.CSS, "\n", "", -1)
		zi.CSS = strings.Replace(zi.CSS, "{ ", "", -1)
		zi.CSS = strings.Replace(zi.CSS, "; ", "", -1)
		findm = strings.Replace(findm, "{ ", "{", -1)
		findm = strings.Replace(findm, "; ", ";", -1)
		if strings.Contains(zi.CSS, findm) || strings.Contains(stripWhitespace(zi.CSS), stripWhitespace(findm)) {
			if i != 0 {
				return errors.New("FindZlib: substring to find is not unique")
			}
			i = zi.Offset
			continue
		}
	}
	if i == 0 {
		return errors.New("FindZlib: could not find string")
	}
	p.cur = i
	return nil
}

// FindZlibHash finds the base address of a zlib css stream based on it's SHA1 hash (can be found using the cssextract tool).
func (p *Patcher) FindZlibHash(hash string) error {
	if len(hash) != 40 {
		return errors.New("FindZlib: invalid hash")
	}
	z, err := p.ExtractZlib()
	if err != nil {
		return errors.Wrap(err, "FindZlib: could not extract zlib streams")
	}
	f := false
	for _, zi := range z {
		if fmt.Sprintf("%x", sha1.Sum([]byte(zi.CSS))) == stripWhitespace(hash) {
			p.cur = zi.Offset
			f = true
			break
		}
	}
	if !f {
		return errors.New("FindZlib: could not find hash")
	}
	return nil
}

// ReplaceZlib replaces a part of a zlib css stream at the current offset.
func (p *Patcher) ReplaceZlib(offset int32, find, replace string) error {
	return p.ReplaceZlibGroup(offset, []Replacement{{find, replace}})
}

// Replacement is a replacement for ReplaceZlibGroup.
type Replacement struct {
	Find, Replace string
}

// ReplaceZlibGroup is the same as ReplaceZlib, but it replaces all at once.
func (p *Patcher) ReplaceZlibGroup(offset int32, repl []Replacement) error {
	if !bytes.HasPrefix(p.buf[p.cur+offset:p.cur+offset+2], []byte{0x78, 0x9c}) {
		return errors.New("ReplaceZlib: not a zlib stream")
	}
	r, err := zlib.NewReader(bytes.NewReader(p.buf[p.cur+offset:])) // Need to use go zlib lib because it is more lenient about corrupt data after end of zlib stream
	if err != nil {
		return errors.Wrap(err, "ReplaceZlib: could not initialize zlib reader")
	}
	dbuf, err := ioutil.ReadAll(r)
	r.Close()
	if err != nil && !strings.Contains(err.Error(), "corrupt input") && !strings.Contains(err.Error(), "invalid checksum") {
		return errors.Wrap(err, "ReplaceZlib: could not decompress stream")
	}
	if len(dbuf) == 0 || !utf8.Valid(dbuf) {
		return errors.New("ReplaceZlib: not a valid zlib stream")
	}
	tbuf := compress(dbuf)
	if !bytes.HasPrefix(p.buf[p.cur+offset:], tbuf) || len(tbuf) < 4 {
		return errors.New("ReplaceZlib: sanity check failed: recompressed original data does not match original (this is a bug, so please report it)")
	}
	for _, r := range repl {
		find, replace := r.Find, r.Replace
		if !bytes.Contains(dbuf, []byte(find)) {
			find = strings.Replace(find, "\n    ", "\n", -1)
			find = strings.Replace(find, "\n  ", "\n", -1)
			find = strings.Replace(find, "\n ", "\n", -1)
			if !bytes.Contains(dbuf, []byte(find)) {
				find = strings.Replace(find, ": ", ":", -1)
				find = strings.Replace(find, " {", "{", -1)
				if !bytes.Contains(dbuf, []byte(find)) {
					find = strings.Replace(find, "\n", "", -1)
					find = strings.Replace(find, "; ", ";", -1)
					find = strings.Replace(find, "{ ", "{", -1)
					if !bytes.Contains(dbuf, []byte(find)) {
						return errors.Errorf("ReplaceZlib: find string not found in stream (%s)", strings.Replace(find, "\n", "\\n", -1))
					}
				}
			}
		}
		dbuf = bytes.Replace(dbuf, []byte(find), []byte(replace), -1)
	}
	nbuf := compress(dbuf)
	if len(nbuf) == 0 {
		return errors.New("ReplaceZlib: error compressing new data (this is a bug, so please report it)")
	}
	if len(nbuf) > len(tbuf) {
		// Attempt to remove indentation to save space
		dbuf = bytes.Replace(dbuf, []byte("\n     "), []byte("\n"), -1)
		dbuf = bytes.Replace(dbuf, []byte("\n  "), []byte("\n"), -1)
		dbuf = bytes.Replace(dbuf, []byte("\n "), []byte("\n"), -1)
		nbuf = compress(dbuf)
	}
	if len(nbuf) > len(tbuf) {
		// Attempt to remove spaces after colons to save space
		dbuf = bytes.Replace(dbuf, []byte(": "), []byte(":"), -1)
		dbuf = bytes.Replace(dbuf, []byte(" {"), []byte("{"), -1)
		nbuf = compress(dbuf)
	}
	if len(nbuf) > len(tbuf) {
		// Attempt to remove newlines to save space
		dbuf = bytes.Replace(dbuf, []byte("\n"), []byte(""), -1)
		dbuf = bytes.Replace(dbuf, []byte("; "), []byte(";"), -1)
		dbuf = bytes.Replace(dbuf, []byte("{ "), []byte("{"), -1)
		nbuf = compress(dbuf)
	}
	if len(nbuf) > len(tbuf) {
		return errors.Errorf("ReplaceZlib: new compressed data is %d bytes longer than old data (try removing whitespace or unnecessary css)", len(nbuf)-len(tbuf))
	}
	copy(p.buf[p.cur+offset:p.cur+offset+int32(len(tbuf))], nbuf)
	r, err = zlib.NewReader(bytes.NewReader(p.buf[p.cur+offset:])) // Need to use go zlib lib because it is more lenient about corrupt data after end of zlib stream
	if err != nil {
		return errors.Wrap(err, "ReplaceZlib: could not initialize zlib reader")
	}
	ndbuf, err := ioutil.ReadAll(r)
	r.Close()
	if !bytes.Equal(dbuf, ndbuf) {
		return errors.New("ReplaceZlib: decompressed new data does not match new data (this is a bug, so please report it)")
	}
	return nil
}

type ZlibItem struct {
	Offset int32
	CSS    string
}

// ExtractZlib extracts all CSS zlib streams. It returns it as a map of offsets and strings.
func (p *Patcher) ExtractZlib() ([]ZlibItem, error) {
	zlibs := []ZlibItem{}
	for i := 0; i < len(p.buf)-2; i++ {
		if bytes.HasPrefix(p.buf[i:i+2], []byte{0x78, 0x9c}) {
			r, err := zlib.NewReader(bytes.NewReader(p.buf[i:])) // Need to use go zlib lib because it is more lenient about corrupt data after end of zlib stream
			if err != nil {
				return zlibs, errors.Wrap(err, "could not initialize zlib reader")
			}
			dbuf, err := ioutil.ReadAll(r)
			r.Close()
			if err != nil && !strings.Contains(err.Error(), "corrupt input") && !strings.Contains(err.Error(), "invalid checksum") {
				return zlibs, errors.Wrap(err, "could not decompress stream")
			}
			if len(dbuf) == 0 || !utf8.Valid(dbuf) {
				continue
			}
			if !isCSS(string(dbuf)) {
				continue
			}
			tbuf := compress(dbuf)
			if !bytes.HasPrefix(p.buf[i:], tbuf) || len(tbuf) < 4 {
				return zlibs, errors.New("sanity check failed: recompressed data does not match original (this is a bug, so please report it)")
			}
			zlibs = append(zlibs, ZlibItem{int32(i), string(dbuf)})
		}
	}
	return zlibs, nil
}

// ReplaceBLX replaces a BLX instruction at PC (offset). Find and Replace are the target offsets.
func (p *Patcher) ReplaceBLX(offset int32, find, replace uint32) error {
	if int32(len(p.buf)) < p.cur+offset {
		return errors.New("offset past end of buf")
	}
	fi, ri := BLX(uint32(p.cur+offset), find), BLX(uint32(p.cur+offset), replace)
	f, r := mustBytes(toBEBin(fi)), mustBytes(toBEBin(ri))
	if len(f) != len(r) {
		return errors.New("internal error: wrong blx length")
	}
	if !bytes.HasPrefix(p.buf[p.cur+offset:], f) {
		return errors.New("could not find bytes")
	}
	copy(p.buf[p.cur+offset:], r)
	return nil
}

// ReplaceBytesNOP replaces an instruction with 0046 (MOV r0, r0) as many times as needed.
func (p *Patcher) ReplaceBytesNOP(offset int32, find []byte) error {
	if int32(len(p.buf)) < offset {
		return errors.New("offset past end of buf")
	}
	if len(find)%2 != 0 {
		return errors.New("find not a multiple of 2")
	}
	r := make([]byte, len(find))
	for i := 0; i < len(r); i += 2 {
		r[i], r[i+1] = 0x00, 0x46
	}
	if !bytes.HasPrefix(p.buf[offset:], find) {
		return errors.New("could not find bytes")
	}
	copy(p.buf[offset:], r)
	return nil
}

// GetCur gets the current base address.
func (p *Patcher) GetCur() int32 {
	return p.cur
}

// replaceValue encodes find and replace as little-endian binary and replaces the first
// occurrence starting at cur. The lengths of the encoded find and replace must be the
// same, or an error will be returned.
func (p *Patcher) replaceValue(offset int32, find, replace interface{}, strictOffset bool) error {
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

	if strictOffset && !bytes.HasPrefix(p.buf[p.cur+offset:], fbuf) {
		fmt.Printf("%x -> %x (%x)[%x]{%d} {%v, %v, %v}\n", fbuf, rbuf, p.buf, p.buf[p.cur+offset:], p.cur+offset, find, replace, offset)
		return errors.New("could not find specified bytes at offset")
	}

	copy(p.buf[p.cur+offset:], bytes.Replace(p.buf[p.cur+offset:], fbuf, rbuf, 1))
	return nil
}

func toLEBin(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, v)
	return buf.Bytes(), err
}

func toBEBin(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, v)
	return buf.Bytes(), err
}

func mustBytes(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}

func wrapErrIfNotNil(txt string, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %v", txt, err)
	}
	return nil
}

func isCSS(str string) bool {
	cob, ccb, cco := strings.Count(str, "{"), strings.Count(str, "}"), strings.Count(str, ":")
	if cob < 1 || ccb < 1 || cco < 1 {
		return false
	}
	if cob != ccb || cob > cco {
		return false
	}
	return true
}

// compress compresses data in a way compatible with python's zlib.
// This uses czlib internally, as the std zlib produces different results.
func compress(src []byte) []byte {
	b, err := czlib.Compress(src) // Need to use czlib to keep header correct
	if err != nil {
		panic(err)
	}
	d, err := decompress(b)
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(d, src) {
		panic("compressed and decompressed data not equal")
	}
	return b
}

func decompress(src []byte) ([]byte, error) {
	return czlib.Decompress(src)
}

func stripWhitespace(src string) string {
	src = strings.Replace(src, " ", "", -1)
	src = strings.Replace(src, "\t", "", -1)
	src = strings.Replace(src, "\n", "", -1)
	src = strings.Replace(src, "\r", "", -1)
	return src
}
