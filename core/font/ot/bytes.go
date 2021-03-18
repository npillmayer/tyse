package ot

import "errors"

/*
We replicate some of the code of the Go core team here, available from
https://github.com/golang/image/tree/master/font/sfnt.
I understand it's legal to do so, as long as the license information stays intact.

We do not use the parsing routines, as they do not fit out purpose, but rather
re-use basic byte-decoding routines, which are not exported. We even simplify
those, as we are always dealing with font data in memory (no io.ReaderAt stuff).

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

The LICENSE file mentioned is replicated as GO-LICENSE at the root directory of
this module.
*/

var errBufferBounds = errors.New("internal inconsistency: buffer bounds error")

func u16(b []byte) uint16 {
	_ = b[1] // Bounds check hint to compiler.
	return uint16(b[0])<<8 | uint16(b[1])<<0
}

func u32(b []byte) uint32 {
	_ = b[3] // Bounds check hint to compiler.
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])<<0
}

// fontBinSegm is a segment of byte data. Conceptually, it is like an io.ReaderAt,
// except that a common segment of SFNT font data is in-memory instead of
// on-disk (e.g. "goregular.TTF") or the result of an ioutil.ReadFile call. In such
// cases, as an optimization, we skip the io.Reader / io.ReaderAt model of
// copying from the fontBinSegm to a caller-supplied buffer, and instead provide
// direct access to the underlying []byte data.
type fontBinSegm []byte

// view returns the length bytes at the given offset.
// The []byte returned is a sub-slice of s.b[]. The caller should not modify the
// contents of the returned []byte
func (b fontBinSegm) view(offset, length int) ([]byte, error) {
	if 0 > offset || offset > offset+length {
		return nil, errBufferBounds
	}
	if offset+length > len(b) {
		return nil, errBufferBounds
	}
	return b[offset : offset+length], nil
}

// varLenView returns bytes from the given offset for sub-tables with varying
// length. The length of bytes is determined by staticLength plus n*itemLength,
// where n is read as uint16 from countOffset (relative to offset).
func (b fontBinSegm) varLenView(offset, staticLength, countOffset, itemLength int) ([]byte, int, error) {
	if 0 > offset || offset > offset+staticLength {
		return nil, 0, errBufferBounds
	}
	if 0 > countOffset || countOffset+1 >= staticLength {
		return nil, 0, errBufferBounds
	}
	// read static part which contains our count
	buf, err := b.view(offset, staticLength)
	if err != nil {
		return nil, 0, err
	}
	count := int(u16(buf[countOffset:]))
	buf, err = b.view(offset, staticLength+count*itemLength)
	if err != nil {
		return nil, 0, err
	}
	return buf, count, nil
}

// u16 returns the uint16 in the table t at the relative offset i.
//
// buf is an optional scratch buffer as per the source.view method.
func (b fontBinSegm) u16(i int) (uint16, error) {
	// //func (b fontBinSegm) u16(t Table, i int) (uint16, error) {
	// 	if i < 0 || uint(t.Len()) < uint(i+2) {
	// 		return 0, errBufferBounds
	// 	}
	// 	buf, err := b.view(int(t.Offset())+i, 2)
	buf, err := b.view(i, 2)
	if err != nil {
		return 0, err
	}
	return u16(buf), nil
}

// u32 returns the uint32 in the table t at the relative offset i.
//
// buf is an optional scratch buffer as per the source.view method.
func (b fontBinSegm) u32(i int) (uint32, error) {
	//func (b fontBinSegm) u32(t Table, i int) (uint32, error) {
	// if i < 0 || uint(t.Len()) < uint(i+4) {
	// 	return 0, errBufferBounds
	// }
	// buf, err := b.view(int(t.Offset())+i, 4)
	buf, err := b.view(i, 4)
	if err != nil {
		return 0, err
	}
	return u32(buf), nil
}