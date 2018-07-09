// Copyright 2016, Datadog Inc.  All rights reserved.

package czlib

import (
	"compress/flate"
	"compress/zlib"
)

// Constants copied from the flate package, so that code that imports czlib
// does not also have to import "compress/flate".
const (
	NoCompression      = flate.NoCompression
	BestSpeed          = flate.BestSpeed
	BestCompression    = flate.BestCompression
	DefaultCompression = flate.DefaultCompression
)

var (
	// ErrChecksum is returned when reading ZLIB data that has an invalid checksum.
	ErrChecksum = zlib.ErrChecksum
	// ErrDictionary is returned when reading ZLIB data that has an invalid dictionary.
	ErrDictionary = zlib.ErrDictionary
	// ErrHeader is returned when reading ZLIB data that has an invalid header.
	ErrHeader = zlib.ErrHeader
)
