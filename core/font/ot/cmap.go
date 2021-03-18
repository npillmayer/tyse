package ot

// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The license file mentioned above can be found as file GO-LICENSE in the
// root folder of this module.
// For further information about licensing please refer to file doc.go in
// this message.

import (
	"fmt"

	"golang.org/x/text/encoding/charmap"
)

// CMapTable represents an OpenType cmap table, i.e. the table to receive glyphs
// from code-points.
type CMapTable struct {
	TableBase
	numTables int
	encRec    encodingRecord
}

type encodingRecord struct {
	platformId uint16
	encodingId uint16
	offset     uint32
	width      int // encoding width
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

type glyphIndexFunc func(otf *OTFont, r rune) (GlyphIndex, error)

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

var supportedCmapFormat = func(format, pid, psid uint16) bool {
	switch format {
	case 0:
		return pid == pidMacintosh && psid == psidMacintoshRoman
	case 4:
		return true
	case 6:
		return true
	case 12:
		return true
	}
	return false
}

func (cmap *CMapTable) makeCachedGlyphIndex(buf []byte, offset, length uint32, format uint16) ([]byte, glyphIndexFunc, error) {
	switch format {
	case 0:
		return cmap.makeCachedGlyphIndexFormat0(offset, length)
	case 4:
		return cmap.makeCachedGlyphIndexFormat4(offset, length)
	case 6:
		return cmap.makeCachedGlyphIndexFormat6(offset, length)
	case 12:
		return cmap.makeCachedGlyphIndexFormat12(offset, length)
	}
	panic("unreachable")
}

func (cmap *CMapTable) makeCachedGlyphIndexFormat0(offset, length uint32) ([]byte, glyphIndexFunc, error) {
	if length != 6+256 || offset+length > cmap.Len() {
		return nil, nil, errFontFormat("invalid cmap size")
	}
	buf, err := cmap.data.view(int(offset), int(length))
	if err != nil {
		return nil, nil, err
	}
	var table [256]byte
	copy(table[:], buf[6:])
	return buf, func(f *OTFont, r rune) (GlyphIndex, error) {
		x, ok := charmap.Macintosh.EncodeRune(r)
		if !ok {
			// The source rune r is not representable in the Macintosh-Roman encoding.
			return 0, nil
		}
		return GlyphIndex(table[x]), nil
	}, nil
}

func (cmap *CMapTable) makeCachedGlyphIndexFormat4(offset, length uint32) ([]byte, glyphIndexFunc, error) {
	const headerSize = 14
	if offset+headerSize > cmap.Len() {
		return nil, nil, errFontFormat("cmap bounds overflow")
	}
	headerdata, err := cmap.data.view(int(offset), headerSize)
	if err != nil {
		return nil, nil, err
	}
	offset += headerSize
	segCount := u16(headerdata[6:])
	if segCount&1 != 0 {
		return nil, nil, errFontFormat("cmap table format")
	}
	segCount /= 2
	if segCount > maxCMapSegments {
		return nil, nil, errFontFormat(fmt.Sprintf("more than %d cmap segments not supported", maxCMapSegments))
	}
	eLength := 8*uint32(segCount) + 2
	if offset+eLength > cmap.Len() {
		return nil, nil, errFontFormat("cmap internal structure")
	}
	segmentsData, err := cmap.data.view(int(offset+offset), int(eLength))
	if err != nil {
		return nil, nil, err
	}
	offset += eLength
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

	return segmentsData, func(otf *OTFont, r rune) (GlyphIndex, error) {
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

func (cmap *CMapTable) makeCachedGlyphIndexFormat6(offset, length uint32) ([]byte, glyphIndexFunc, error) {
	const headerSize = 10
	if offset+headerSize > cmap.Len() {
		return nil, nil, errFontFormat("cmap bounds overflow")
	}
	buf, err := cmap.data.view(int(offset), headerSize)
	if err != nil {
		return nil, nil, err
	}
	offset += headerSize

	firstCode := u16(buf[6:])
	entryCount := u16(buf[8:])

	eLength := 2 * uint32(entryCount)
	if offset+eLength > cmap.Len() {
		return nil, nil, errFontFormat("cmap bounds overflow")
	}

	if entryCount != 0 {
		buf, err = cmap.data.view(int(offset), int(eLength))
		if err != nil {
			return nil, nil, err
		}
		offset += eLength
	}

	entries := make([]uint16, entryCount)
	for i := range entries {
		entries[i] = u16(buf[2*i:])
	}

	return buf, func(otf *OTFont, r rune) (GlyphIndex, error) {
		if uint16(r) < firstCode {
			return 0, nil
		}
		c := int(uint16(r) - firstCode)
		if c >= len(entries) {
			return 0, nil
		}
		return GlyphIndex(entries[c]), nil
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

	return buf, func(otf *OTFont, r rune) (GlyphIndex, error) {
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

type cmapEntry16 struct {
	end, start, delta, offset uint16
}

type cmapEntry32 struct {
	start, end, delta uint32
}
