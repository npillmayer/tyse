package ot

/*
We replicate some of the code of the Go core team here, available from
https://github.com/golang/image/tree/master/font/sfnt.
I understand it's legal to do so, as long as the license information stays intact.

   Copyright 2017 The Go Authors. All rights reserved.
   Use of this source code is governed by a BSD-style
   license that can be found in the LICENSE file.

The LICENSE file mentioned is replicated as GO-LICENSE at the root directory of
this module.
*/

import (
	"fmt"
)

// CMapTable represents an OpenType cmap table, i.e. the table to receive glyphs
// from code-points.
//
// See https://docs.microsoft.com/de-de/typography/opentype/spec/cmap
//
type CMapTable struct {
	TableBase
	GlyphIndexMap CMapGlyphIndex
}

// type encodingRecord struct {
// 	platformId uint16
// 	encodingId uint16
// 	offset     uint32
// 	width      int // encoding width
// }

func newCMapTable(tag Tag, b fontBinSegm, offset, size uint32) *CMapTable {
	t := &CMapTable{}
	base := TableBase{
		data:   b,
		name:   tag,
		offset: offset,
		length: size,
	}
	t.TableBase = base
	t.self = t
	return t
}

func (t *CMapTable) Base() *TableBase {
	return &t.TableBase
}

// Platform IDs and Platform Specific IDs as per
// https://www.microsoft.com/typography/otspec/name.htm
const (
	pidUnicode   = 0
	pidMacintosh = 1
	pidWindows   = 3

	// Note that FontForge may generate a bogus Platform Specific ID (value 10)
	// for the Unicode Platform ID (value 0). See
	// https://github.com/fontforge/fontforge/issues/2728
	psidUnicode2BMPOnly        = 3
	psidUnicode2FullRepertoire = 4
	psidMacintoshRoman         = 0
	psidWindowsSymbol          = 0
	psidWindowsUCS2            = 1
	psidWindowsUCS4            = 10
)

const (
	// This value is arbitrary, but defends against parsing malicious font
	// files causing excessive memory allocations. For reference, Adobe's
	// SourceHanSansSC-Regular.otf has 65535 glyphs and:
	//	- its format-4  cmap table has  1581 segments.
	//	- its format-12 cmap table has 16498 segments.
	maxCMapSegments = 20000

	// Adobe's SourceHanSansSC-Regular.otf has up to 30000 subroutines.
	maxNumSubroutines = 40000
)

type glyphIndexFunc func(otf *Font, r rune) (GlyphIndex, error)

// platformEncodingWidth returns the number of bytes per character assumed by
// the given Platform ID and Platform Specific ID.
//
// Very old fonts, from before Unicode was widely adopted, assume only 1 byte
// per character: a character map.
//
// Old fonts, from when Unicode meant the Basic Multilingual Plane (BMP),
// assume that 2 bytes per character is sufficient.
//
// Recent fonts naturally support the full range of Unicode code points, which
// can take up to 4 bytes per character. Such fonts might still choose one of
// the legacy encodings if e.g. their repertoire is limited to the BMP, for
// greater compatibility with older software, or because the resultant file
// size can be smaller.
func platformEncodingWidth(pid, psid uint16) int {
	switch pid {
	case pidUnicode:
		switch psid {
		case psidUnicode2BMPOnly:
			return 2
		case psidUnicode2FullRepertoire:
			return 4
		}

	case pidMacintosh:
		switch psid {
		case psidMacintoshRoman:
			return 1
		}

	case pidWindows:
		switch psid {
		case psidWindowsSymbol:
			return 2
		case psidWindowsUCS2:
			return 2
		case psidWindowsUCS4:
			return 4
		}
	}
	return 0
}

// The various cmap formats are described at
// https://www.microsoft.com/typography/otspec/cmap.htm

// All in all, we only support the following plaform/encoding/format combinations:
//   0 (Unicode)  3    4   Unicode BMB
//   0 (Unicode)  4    12  Unicode full  (10 from FontForge, error)
//   3 (Win)      1    4   Unicode BMP
//   3 (Win)      10   12  Unicode full
//
// Note that FontForge may generate a bogus Platform Specific ID (value 10)
// for the Unicode Platform ID (value 0). See
// https://github.com/fontforge/fontforge/issues/2728
var supportedCmapFormat = func(format, pid, psid uint16) bool {
	return (pid == 0 && psid == 3 && format == 4) ||
		(pid == 0 && psid == 4 && format == 12) ||
		(pid == 3 && psid == 1 && format == 4) ||
		(pid == 3 && psid == 10 && format == 12)
}

func (cmap *CMapTable) makeCachedGlyphIndex(buf []byte, offset, length uint32, format uint16) ([]byte, glyphIndexFunc, error) {
	switch format {
	case 4:
		return cmap.makeCachedGlyphIndexFormat4(offset, length)
	case 12:
		return cmap.makeCachedGlyphIndexFormat12(offset, length)
	}
	panic("unreachable")
}

// CMapGlyphIndex represents a CMap table index to receive a glyph index from
// a code-point.
type CMapGlyphIndex interface {
	GlyphIndex(rune) GlyphIndex
}

// Format 4: Segment mapping to delta values
// This is the standard character-to-glyph-index mapping subtable for fonts that support
// only Unicode Basic Multilingual Plane characters (U+0000 to U+FFFF).
//
// This format is used when the character codes for the characters represented by a font
// fall into several contiguous ranges, possibly with holes in some or all of the ranges
// (that is, some of the codes in a range may not have a representation in the font).
// The format-dependent data is divided into three parts, which must occur in the following
// order:
// - A four-word header gives parameters for an optimized search of the segment list;
// - Four parallel arrays describe the segments (one segment for each contiguous range of codes);
// - A variable-length array of glyph IDs (unsigned words).
//
type cmapEntry16 struct {
	end, start, delta, offset uint16
}

func (cmap *CMapTable) makeCachedGlyphIndexFormat4(b fontBinSegm) (glyphIndexFunc, error) {
	const headerSize = 14
	if headerSize > b.Size() {
		return nil, errFontFormat("cmap subtable bounds overflow")
	}
	//size, _ := b.u16(2)
	segCount, _ := b.u16(6)
	if segCount&1 != 0 {
		return nil, errFontFormat("cmap table format, illegal segment count")
	}
	segCount /= 2
	eLength := 8*int(segCount) + 2
	if eLength > b.Size() {
		return nil, errFontFormat("cmap internal structure")
	}
	segmentsData, err := b.view(headerSize, eLength)
	if err != nil {
		return nil, err
	}
	entries := make([]cmapEntry16, segCount)
	for i := range entries {
		entries[i] = cmapEntry16{
			end:    u16(segmentsData[0*len(entries)+0+2*i:]),
			start:  u16(segmentsData[2*len(entries)+2+2*i:]),
			delta:  u16(segmentsData[4*len(entries)+2+2*i:]),
			offset: u16(segmentsData[6*len(entries)+2+2*i:]),
		}
	}
	indexesBase := offset
	indexesLength := cmap.Len() - offset

	return segmentsData, func(otf *Font, r rune) (GlyphIndex, error) {
		if uint32(r) > 0xffff {
			return 0, nil
		}

		c := uint16(r)
		for i, j := 0, len(entries); i < j; {
			h := i + (j-i)/2
			entry := &entries[h]
			if c < entry.start {
				j = h
			} else if entry.end < c {
				i = h + 1
			} else if entry.offset == 0 {
				return GlyphIndex(c + entry.delta), nil
			} else {
				offset := uint32(entry.offset) + 2*uint32(h-len(entries)+int(c-entry.start))
				if offset > indexesLength || offset+2 > indexesLength {
					return 0, errFontFormat("cmap bounds overflow")
				}
				x, err := cmap.data.view(int(indexesBase+offset), 2)
				if err != nil {
					return 0, err
				}
				return GlyphIndex(u16(x)), nil
			}
		}
		return 0, nil
	}, nil
}

func (cmap *CMapTable) makeCachedGlyphIndexFormat12(offset, _ uint32) ([]byte, glyphIndexFunc, error) {
	const headerSize = 16
	if offset+headerSize > cmap.Len() {
		return nil, nil, errFontFormat("cmap bounds overflow")
	}
	buf, err := cmap.data.view(int(offset), headerSize)
	if err != nil {
		return nil, nil, err
	}
	length := u32(buf[4:])
	if cmap.Len() < offset || length > cmap.Len()-offset {
		return nil, nil, errFontFormat("cmap bounds overflow")
	}
	offset += headerSize
	numGroups := u32(buf[12:])
	if numGroups > maxCMapSegments {
		return nil, nil, errFontFormat(fmt.Sprintf("more than %d cmap segments not supported", maxCMapSegments))
	}
	eLength := 12 * numGroups
	if headerSize+eLength != length {
		return nil, nil, errFontFormat("cmap table format")
	}
	buf, err = cmap.data.view(int(offset), int(eLength))
	if err != nil {
		return nil, nil, err
	}
	offset += eLength
	entries := make([]cmapEntry32, numGroups)
	for i := range entries {
		entries[i] = cmapEntry32{
			start: u32(buf[0+12*i:]),
			end:   u32(buf[4+12*i:]),
			delta: u32(buf[8+12*i:]),
		}
	}

	return buf, func(otf *Font, r rune) (GlyphIndex, error) {
		c := uint32(r)
		for i, j := 0, len(entries); i < j; {
			h := i + (j-i)/2
			entry := &entries[h]
			if c < entry.start {
				j = h
			} else if entry.end < c {
				i = h + 1
			} else {
				return GlyphIndex(c - entry.start + entry.delta), nil
			}
		}
		return 0, nil
	}, nil
}

type cmapEntry32 struct {
	start, end, delta uint32
}
