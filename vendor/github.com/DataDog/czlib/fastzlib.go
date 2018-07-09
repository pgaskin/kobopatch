// Copyright 2013, Datadog Inc.  All rights reserved.

package czlib

import (
	"errors"
	"unsafe"
)

/*
#cgo LDFLAGS: -lz
#include "fastzlib.h"
#include <stdlib.h>
*/
import "C"

// An UnsafeByte is a []byte whose backing array has been allocated in C and
// thus is not subject to the Go garbage collector.  The Unsafe versions of
// Compress and Decompress return this in order to prevent copying the unsafe
// memory into collected memory.
type UnsafeByte []byte

// NewUnsafeByte creates a []byte from the unsafe pointer without a copy,
// using the method outlined in this mailing list post:
//   https://groups.google.com/forum/#!topic/golang-nuts/KyXR0fDp0HA
// but amended to use the three-index slices from go1.2 to set the capacity
// of b correctly:
//   https://tip.golang.org/doc/go1.2#three_index
// This means this code only works in go1.2+.
//
// This shouldn't copy the underlying array;  it's just casting it
// Afterwards, we use reflect to fix the Cap & len of the slice.
func NewUnsafeByte(p *C.char, length int) UnsafeByte {
	var b UnsafeByte
	b = UnsafeByte((*[1<<31 - 1]byte)(unsafe.Pointer(p))[:length:length])
	return b
}

// Free the underlying byte array;  doing this twice would be bad.
func (b UnsafeByte) Free() {
	C.free(unsafe.Pointer(&b[0]))
}

// Compress returns the input compressed using zlib, or an error if encountered.
func Compress(input []byte) ([]byte, error) {
	var cInput *C.char
	if len(input) != 0 {
		cInput = (*C.char)(unsafe.Pointer(&input[0]))
	}
	ret := C.c_compress2(cInput, C.uint(len(input)))

	// if there was an error compressing, return it and free the original message
	if ret.err != nil {
		msg := C.GoString((*C.char)(ret.err))
		C.free(unsafe.Pointer(ret.err))
		return []byte{}, errors.New(msg)
	}

	// NOTE: this creates a copy of the return *char as a Go []byte.
	// FIXME: uint -> int conversion here is dangerous
	b := C.GoBytes(unsafe.Pointer(ret.str), C.int(ret.len))
	C.free(unsafe.Pointer(ret.str))
	return b, nil
}

// Decompress returns the input decompressed using zlib, or an error if encountered.
func Decompress(input []byte) ([]byte, error) {
	var cInput *C.char
	if len(input) != 0 {
		cInput = (*C.char)(unsafe.Pointer(&input[0]))
	}
	// send the input byte without copying iy
	ret := C.c_decompress(cInput, C.uint(len(input)))

	// if there was an error decompressing, return it and free the original message
	if ret.err != nil {
		msg := C.GoString((*C.char)(ret.err))
		C.free(unsafe.Pointer(ret.err))
		return []byte{}, errors.New(msg)
	}

	// NOTE: this creates a copy of the return *char as a Go []byte.
	// FIXME: uint -> int conversion here is dangerous
	b := C.GoBytes(unsafe.Pointer(ret.str), C.int(ret.len))
	C.free(unsafe.Pointer(ret.str))
	return b, nil
}

// UnsafeDecompress unzips input into an UnsafeByte without copying the result
// malloced in C.  The UnsafeByte returned can be used as a normal []byte but
// must be manually free'd w/ UnsafeByte.Free()
func UnsafeDecompress(input []byte) (UnsafeByte, error) {
	cInput := (*C.char)(unsafe.Pointer(&input[0]))
	ret := C.c_decompress(cInput, C.uint(len(input)))

	// if there was an error decompressing, return it and free the original message
	if ret.err != nil {
		msg := C.GoString((*C.char)(ret.err))
		C.free(unsafe.Pointer(ret.err))
		return UnsafeByte{}, errors.New(msg)
	}

	b := NewUnsafeByte((*C.char)(ret.str), int(ret.len))
	return b, nil
}

// UnsafeCompress zips input into an UnsafeByte without copying the result
// malloced in C.  The UnsafeByte returned can be used as a normal []byte but must
// be manually free'd w/ UnsafeByte.Free()
func UnsafeCompress(input []byte) (UnsafeByte, error) {
	cInput := (*C.char)(unsafe.Pointer(&input[0]))
	ret := C.c_compress(cInput, C.uint(len(input)))

	// if there was an error decompressing, return it and free the original message
	if ret.err != nil {
		msg := C.GoString((*C.char)(ret.err))
		C.free(unsafe.Pointer(ret.err))
		return UnsafeByte{}, errors.New(msg)
	}

	b := NewUnsafeByte((*C.char)(ret.str), int(ret.len))
	return b, nil
}
