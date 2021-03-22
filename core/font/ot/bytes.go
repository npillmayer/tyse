package ot

import (
	"bytes"
	"errors"
	"io"
)

// Reading bytes from a font's binary representation

var errBufferBounds = errors.New("internal inconsistency: buffer bounds error")

func u16(b []byte) uint16 {
	_ = b[1] // Bounds check hint to compiler.
	return uint16(b[0])<<8 | uint16(b[1])<<0
}

func u32(b []byte) uint32 {
	_ = b[3] // Bounds check hint to compiler.
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])<<0
}

// ---Locations, i.e. byte segments/slices -----------------------------------

// Location is a position at a byte within a font's binary data.
// It represents the start of a segment/slice of binary data.
type Location interface {
	Size() int     // size in bytes
	Bytes() []byte // return as a byte slice
	Reader() io.Reader
	U16() uint16
	U32() uint32
}

// fontBinSegm is a segment of byte data.
// It implements the Location interface. We use it throughout in this module to
// naviagte the font's binary data.
type fontBinSegm []byte

func (b fontBinSegm) Size() int {
	return len(b)
}

func (b fontBinSegm) Bytes() []byte {
	return b
}

func (b fontBinSegm) Reader() io.Reader {
	return bytes.NewReader(b)
}

func (b fontBinSegm) U16() uint16 {
	return u16(b)
}

func (b fontBinSegm) U32() uint32 {
	return u32(b)
}

// view returns n bytes at the given offset.
// The byte segment returned is a sub-slice of b.
func (b fontBinSegm) view(offset, n int) (fontBinSegm, error) {
	if offset < 0 || n <= 0 || offset+n > len(b) {
		return nil, errBufferBounds
	}
	return b[offset : offset+n], nil
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

// u16 returns the uint16 in b at the relative offset i.
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

// u32 returns the uint32 in b at the relative offset i.
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

// --- Ranges of glyphs ------------------------------------------------------

type GlyphRange interface {
	Lookup(g rune) (int, bool)
	ByteSize() int
}

type glyphRangeArray struct {
	is32     bool // keys are 32 bit
	count    int  // number of glyph keys
	data     fontBinSegm
	byteSize int
}

// glyphRangeArrays have entries stored as a block of consecutive keys.
// glyphRangeArrays return the index of the key in the range table.
// 0 is a valid return value.
func (r *glyphRangeArray) Lookup(g rune) (int, bool) {
	if r.count <= 0 {
		return 0, false
	}
	if r.is32 {
		for i := 0; i < r.count; i++ {
			k, err := r.data.u32(i * 4)
			if err != nil {
				return 0, false
			} else if rune(k) == g {
				return i, true
			}
		}
	} else {
		for i := 0; i < r.count; i++ {
			k, err := r.data.u16(i * 2)
			if err != nil {
				return 0, false
			} else if rune(k) == g {
				return i, true
			}
		}
	}
	return 0, false
}

type rangeRecord struct {
	from, to rune
	index    uint16
}

func (r *glyphRangeArray) ByteSize() int {
	return r.byteSize
}

type glyphRangeRecords struct {
	is32     bool // keys are 32 bit
	count    int  // number of range records
	data     fontBinSegm
	byteSize int
}

// glyphRangeRecords have entries stored as range records.
// glyphRangeRecords return the index of the key in the range table.
// 0 is a valid return value.
func (r *glyphRangeRecords) Lookup(g rune) (int, bool) {
	if r.count <= 0 {
		return 0, false
	}
	record := rangeRecord{}
	if r.is32 {
		for i := 0; i < r.count; i++ {
			k, err := r.data.u32(i * (4 + 4 + 2))
			if err != nil {
				return 0, false
			}
			record.from = rune(k)
			k, _ = r.data.u32(i*(2+2+2) + 4)
			record.to = rune(k)
			v, _ := r.data.u16(i*(2+2+2) + 6)
			record.index = v
		}
	} else {
		for i := 0; i < r.count; i++ {
			k, err := r.data.u16(i * (2 + 2 + 2))
			if err != nil {
				return 0, false
			}
			record.from = rune(k)
			k, _ = r.data.u16(i*(2+2+2) + 2)
			record.to = rune(k)
			k, _ = r.data.u16(i*(2+2+2) + 4)
			record.index = k
		}
	}
	return 0, false
}

func (r *glyphRangeRecords) ByteSize() int {
	return r.byteSize
}

// --- Tag list --------------------------------------------------------------

type tagList struct {
	Count int
	link  Link
}

func parseTagList(b fontBinSegm) tagList {
	tl := tagList{Count: int(u16(b))}
	tl.link = link16{
		base:   b,
		offset: 2,
	}
	return tl
}

func (l tagList) Tag(i int) Tag {
	const taglen = 4
	if b := l.link.Jump(); len(b.Bytes()) >= (i+1)*taglen {
		if n, err := fontBinSegm(b.Bytes()).u32(i * taglen); err == nil {
			return Tag(n)
		}
	}
	return Tag(0)
}

// --- Link ------------------------------------------------------------------

type Link interface {
	Base() Location
	Jump() Location
	IsNull() bool
}

func parseLink32(b fontBinSegm, offset int, base fontBinSegm) (Link, error) {
	if len(b) < offset {
		return link32{}, errBufferBounds
	}
	n, err := b.u32(offset)
	if err != nil {
		return link32{}, errBufferBounds
	}
	return link32{
		base:   base,
		offset: n,
	}, nil
}

func parseLink16(b fontBinSegm, offset int, base fontBinSegm) (Link, error) {
	if len(b) < offset {
		return link16{}, errBufferBounds
	}
	n, err := b.u16(offset)
	if err != nil {
		return link16{}, errBufferBounds
	}
	return link16{
		base:   base,
		offset: n,
	}, nil
}

type link16 struct {
	base   fontBinSegm
	offset uint16
}

func (l16 link16) IsNull() bool {
	return len(l16.base) == 0
}

func (l16 link16) Base() Location {
	return l16.base
}

func (l16 link16) Jump() Location {
	if l16.offset > uint16(len(l16.base)) {
		return fontBinSegm{}
	}
	return l16.base[l16.offset:]
}

type link32 struct {
	base   fontBinSegm
	offset uint32
}

func (l32 link32) IsNull() bool {
	return len(l32.base) == 0
}

func (l32 link32) Base() Location {
	return l32.base
}

func (l32 link32) Jump() Location {
	if l32.offset > uint32(len(l32.base)) {
		trace().Debugf("base has size %d", len(l32.base))
		trace().Debugf("link to %d", l32.offset)
		trace().Debugf("offset32 location out of table bounds")
		return fontBinSegm{}
	}
	return l32.base[l32.offset:]
}

// --- Arrays ----------------------------------------------------------------

type array struct {
	recordSize int
	length     int
	loc        fontBinSegm
}

func parseArrary32(b fontBinSegm) array {
	if b.Size()&0x11 != 0 {
		trace().Errorf("cannot create array32: size not aligned")
		return array{}
	}
	n := b.Size() / 4
	return array{
		recordSize: 4,
		length:     n,
		loc:        b,
	}
}

func parseArray16(b fontBinSegm) array {
	if b.Size()&0x1 != 0 {
		trace().Errorf("cannot create array16: size not aligned")
		return array{}
	}
	n := b.Size() / 2
	return array{
		recordSize: 2,
		length:     n,
		loc:        b,
	}
}

func parseArray(b fontBinSegm, recordSize int) array {
	return array{
		recordSize: recordSize,
		length:     b.Size() / recordSize,
		loc:        b,
	}
}

func (a array) Size() int {
	return a.length * a.recordSize
}

func (a array) UnsafeGet(i int) Location {
	if i < 0 || (i+1)*a.recordSize > len(a.loc.Bytes()) {
		i = 0
	}
	b, _ := a.loc.view(i*a.recordSize, a.recordSize)
	return b
}

// --- Tag record map --------------------------------------------------------

// A TagRecordMap is a dict-typ (map) to receive a data record (returned as a link)
// from a given tag. This kind of map is used within OpenType fonts in several
// instances, e.g.
// https://docs.microsoft.com/en-us/typography/opentype/spec/base#basescriptlist-table
type TagRecordMap interface {
	Lookup(Tag) Link
}

func parseTagRecordMap16(b fontBinSegm, offset, recsize int) TagRecordMap {
	size, err := b.u16(offset)
	if err != nil {
		return tagRecordMap16{}
	}
	m := tagRecordMap16{base: b[offset+2 : offset+2+int(size)], recordSize: recsize}
	return m
}

type tagRecordMap16 struct {
	base       fontBinSegm
	recordSize int
}

func (m tagRecordMap16) Lookup(tag Tag) Link {
	if m.recordSize <= 0 {
		return link16{}
	}
	// TODO
	panic("TagRecordMap.Lookup() not yet implemented")
	//return nil
}
