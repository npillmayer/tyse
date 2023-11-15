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

// CMapTable represents an OpenType cmap table, i.e. the table to receive glyphs
// from code-points.
//
// See https://docs.microsoft.com/de-de/typography/opentype/spec/cmap
//
// Consulting the cmap table is a very frequent operation on fonts. We therefore
// construct an internal representation of the lookup table. A cmap table may contain
// more than one lookup table, but we will only instantiate the most appropriate one.
// Clients who need access to all the lookup tables will have to parse them themselves.
type CMapTable struct {
	tableBase
	GlyphIndexMap CMapGlyphIndex
}

func newCMapTable(tag Tag, b binarySegm, offset, size uint32) *CMapTable {
	t := &CMapTable{}
	base := tableBase{
		data:   b,
		name:   tag,
		offset: offset,
		length: size,
	}
	t.tableBase = base
	t.self = t
	return t
}

// platformEncodingWidth returns the number of bytes per character assumed by
// the given Platform ID and Platform Specific ID.
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
	case 0: // Unicode platform
		switch psid {
		case 3: // Unicode BMB
			return 2
		case 4, 10: // Unicode full  (include 10 from FontForge bug)
			return 4
		}
	case 3: // Windows platform
		switch psid {
		case 1: // Unicode BMP
			return 2
		case 10: // Unicode full
			return 4
		}
	}
	return 0 // width 0 will never get selected
}

// The various cmap formats are described at
// https://www.microsoft.com/typography/otspec/cmap.htm
//
// From the spec.: Of the seven available formats, not all are commonly used today.
// Formats 4 or 12 are appropriate for most new fonts, depending on the Unicode character
// repertoire supported. Format 14 is used in many applications for support of Unicode
// variation sequences. Some platforms also make use for format 13 for a last-resort
// fallback font. Other subtable formats are not recommended for use in new fonts.
// Application developers, however, should anticipate that any of the formats may be used
// in fonts.
//
// Right now we do not support variable fonts nor fallback fonts.
// All in all, we only support the following plaform/encoding/format combinations:
//
//	0 (Unicode)  3    4   Unicode BMB
//	0 (Unicode)  4    12  Unicode full  (10 from FontForge, error)
//	3 (Win)      1    4   Unicode BMP
//	3 (Win)      10   12  Unicode full
//
// Note that FontForge may generate a bogus Platform Specific ID (value 10)
// for the Unicode Platform ID (value 0). See
// https://github.com/fontforge/fontforge/issues/2728
func supportedCmapFormat(format, pid, psid uint16) bool {
	tracer().Debugf("checking supported cmap format (%d | %d | %d)", pid, psid, format)
	return (pid == 0 && psid == 3 && format == 4) ||
		(pid == 0 && psid == 4 && format == 12) ||
		(pid == 3 && psid == 1 && format == 4) ||
		(pid == 3 && psid == 10 && format == 12)
}

// Dispatcher to create the correct implementation of a CMapGlyphIndex from a given format.
func makeGlyphIndex(b binarySegm, which encodingRecord) (CMapGlyphIndex, error) {
	subtable := which.link.Jump()
	switch which.format {
	case 4:
		return makeGlyphIndexFormat4(subtable.Bytes())
	case 12:
		return makeGlyphIndexFormat12(subtable.Bytes())
	}
	panic("unreachable") // unsupported formats should have been weeded out beforehand
}

// CMapGlyphIndex represents a CMap table index to receive a glyph index from
// a code-point.
type CMapGlyphIndex interface {
	Lookup(rune) GlyphIndex        // central activiy of CMap
	ReverseLookup(GlyphIndex) rune // this is non-standard, but helps with tests
}

// Format 4: Segment mapping to delta values
// This is the standard character-to-glyph-index mapping subtable for fonts that support
// only Unicode Basic Multilingual Plane characters (U+0000 to U+FFFF).
//
// This format is used when the character codes for the characters represented by a font
// fall into several contiguous ranges, possibly with holes in some or all of the ranges
// (that is, some of the codes in a range may not have a representation in the font).
type format4GlyphIndex struct {
	segCnt   int
	entries  []cmapEntry16
	glyphIds array
}

// Format 4 holds four parallel arrays to describe the segments (one segment for
// each contiguous range of codes).
// see https://docs.microsoft.com/en-us/typography/opentype/spec/cmap#format-4-segment-mapping-to-delta-values
type cmapEntry16 struct {
	end, start, delta, offset uint16
}

func (f4 format4GlyphIndex) Lookup(r rune) GlyphIndex {
	if uint32(r) > 0xffff { // format 4 is for BMP code-points only
		return 0 // return index for 'missing character'
	}
	c := uint16(r)
	N := len(f4.entries)
	//trace().Debugf("lookup codepoint %d in %d cmap-ranges", r, N)
	for i, j := 0, N; i < j; {
		h := i + (j-i)/2 // do a binary search on f4.entries (which may get large)
		entry := &f4.entries[h]
		if c < entry.start {
			j = h
		} else if entry.end < c {
			i = h + 1
		} else if entry.offset == 0 {
			//tracer().Debugf("direct access of glyph ID as delta = %d", c+entry.delta)
			return GlyphIndex(c + entry.delta)
		} else {
			// The spec describes the calculation the find the link into the glyph ID array
			// as follows:
			// “The character code offset from startCode is added to the idRangeOffset value.
			//  This sum is used as an offset from the current location within idRangeOffset
			//  itself to index out the correct glyphIdArray value. This obscure indexing
			//  trick works because glyphIdArray immediately follows idRangeOffset in the
			//  font file.”
			// We already sliced the cmap into sub-segments, so this will not work for us
			// (intentionally–I'm not a big fan of 'obscure' tricks). Instead, we will
			// calculate a clean index into the glyph ID array. Unfortunately this requires
			// us to reverse some of the magic pre-calculations in the font—a procedure which
			// one may consider obscure as well, but that's life…
			//
			// First cut off the part off the trailing part of offset which results from
			// skipping over to the start of the glyph ID array:
			//
			// --- for now leave traces in as next bug will surely wait...
			// eprev := &f4.entries[h-1]
			// trace().Debugf("segment #%d = { start=%d, end=%d, delta=%d, offset=%d }",
			// 	h-1, eprev.start, eprev.end, eprev.delta, eprev.offset)
			// enext := &f4.entries[h+1]
			// trace().Debugf("segment #%d = { start=%d, end=%d, delta=%d, offset=%d }",
			// 	h+1, enext.start, enext.end, enext.delta, enext.offset)
			deltaToEndOfEntries := (N - h) * 2 // 2 = byte size of offset array entry
			//trace().Debugf("N = %d, N*2 = %d, h = %d, h*2=%d", N, N*2, h, h*2)
			offset := int(entry.offset) - deltaToEndOfEntries
			// Now normalize the index into the glyph ID array
			index := offset / 2 // offset is in bytes, we need an array index
			index += int(c - entry.start)
			glyphInx := f4.glyphIds.Get(index).U16(0)
			// trace().Debugf("segment #%d = { start=%d, end=%d, delta=%d, offset=%d }",
			// 	h, entry.start, entry.end, entry.delta, entry.offset)
			// trace().Debugf("skip = %d, offset = %d, rest = %d", deltaToEndOfEntries, offset, index)
			// trace().Debugf("looking up code-point in segment %d, is %d", h, glyphInx)
			if glyphInx > 0 {
				// If the value obtained from the indexing operation is not 0 (which indicates
				// missingGlyph), idDelta[i] is added to it to get the glyph index
				glyphInx += entry.delta
			}
			// g2 := f4.glyphIds.UnsafeGet(index + 1).U16(0)
			// trace().Debugf("next glyph ID = %d", g2)
			// g2 = f4.glyphIds.UnsafeGet(index + 2).U16(0)
			// trace().Debugf("next glyph ID = %d", g2)
			// g2 = f4.glyphIds.UnsafeGet(index + 3).U16(0)
			// trace().Debugf("next glyph ID = %d", g2)
			return GlyphIndex(glyphInx) // will be 0 in case of indexing error
		}
	}
	return GlyphIndex(0)
}

// ReverseLookup retrieves a code-point for a given glyph. The Cmap tables do not
// support this operation, thus this operation is inefficient.
// However, for testing and debugging purposes it is often useful.
func (f4 format4GlyphIndex) ReverseLookup(gid GlyphIndex) rune {
	if gid == 0 {
		return 0
	}
	//tracer().Debugf("CMap format 4 has %d entries", len(f4.entries))
	for _, entry := range f4.entries {
		if entry.end < entry.start || entry.start == 0xffff {
			break
		}
		//tracer().Debugf("CMap format 4 entry: %d … %d", entry.start, entry.end)
		for c := entry.start; c <= entry.end; c++ {
			if f4.Lookup(rune(c)) == gid {
				return rune(c)
			}
		}
	}
	return 0
}

// The format's data is divided into three parts, which must occur in the following order:
//
// - A four-word header gives parameters for an optimized search of the segment list;
// - Four parallel arrays describe the segments (one segment for each contiguous range of codes);
// - A variable-length array of glyph IDs (unsigned words).
func makeGlyphIndexFormat4(b binarySegm) (CMapGlyphIndex, error) {
	const headerSize = 14
	if headerSize > b.Size() {
		return nil, errFontFormat("cmap subtable bounds overflow")
	}
	size, _ := b.u16(2)
	segCount, _ := b.u16(6)
	if segCount&1 != 0 {
		tracer().Debugf("cmap format 4 segment count is %d", segCount)
		return nil, errFontFormat("cmap table format, illegal segment count")
	}
	segCount /= 2
	eLength := 8*int(segCount) + 2
	if eLength > b.Size() || headerSize+eLength > int(size) {
		return nil, errFontFormat("cmap internal structure")
	}
	b = b[headerSize:size]
	endCodes := viewArray16(b[:segCount*2])
	next := endCodes.Size() + 2 // 2 is a padding entry in the cmap table
	startCodes := viewArray16(b[next : next+int(segCount)*2])
	next += startCodes.Size()
	deltas := viewArray16(b[next : next+int(segCount)*2])
	next += deltas.Size()
	offsets := viewArray16(b[next : next+int(segCount)*2])
	next += offsets.Size()
	entries := make([]cmapEntry16, segCount)
	for i := range entries {
		entries[i] = cmapEntry16{
			end:    endCodes.Get(i).U16(0),
			start:  startCodes.Get(i).U16(0),
			delta:  deltas.Get(i).U16(0),
			offset: offsets.Get(i).U16(0),
		}
		if entries[i].offset > 0 && entries[i].delta > 0 {
			panic("Hurray! Font with cmap format 4, offset > 0 and delta > 0, detected!")
		}
	}
	glyphTable := viewArray16(b[next:])
	tracer().Debugf("cmap format 4 glyph table starts at offset %d", next)
	return format4GlyphIndex{
		segCnt:   int(segCount),
		entries:  entries,
		glyphIds: glyphTable,
	}, nil
}

type cmapEntry32 struct {
	start, end, delta uint32
}

// Each sequential map group record specifies a character range and the starting glyph ID
// mapped from the first character. Glyph IDs for subsequent characters follow in sequence.
type format12GlyphIndex struct {
	grpCnt  int
	entries []cmapEntry32
}

func (f12 format12GlyphIndex) Lookup(r rune) GlyphIndex {
	c := uint32(r)
	for i, j := 0, len(f12.entries); i < j; {
		h := i + (j-i)/2 // do a binary search on f12.entries (which may get large)
		entry := &f12.entries[h]
		if c < entry.start {
			j = h
		} else if entry.end < c {
			i = h + 1
		} else {
			return GlyphIndex(c - entry.start + entry.delta)
		}
	}
	return 0
}

// ReverseLookup retrieves a code-point for a given glyph. The Cmap tables do not
// support this operation, thus this operation is inefficient.
// However, for testing and debugging purposes it is often useful.
func (f12 format12GlyphIndex) ReverseLookup(gid GlyphIndex) rune {
	if gid == 0 {
		return 0
	}
	cid := uint32(gid)
	for _, entry := range f12.entries {
		for c := entry.start; c <= entry.end; c++ {
			if c-entry.start+entry.delta == cid {
				return rune(c)
			}
		}
	}
	return 0
}

// This is the standard character-to-glyph-index mapping subtable for fonts supporting
// Unicode character repertoires that include supplementary-plane characters (U+10000 to
// U+10FFFF).
//
// Format 12 is similar to format 4 in that it defines segments for sparse representation.
// It differs, however, in that it uses 32-bit character codes, and Glyph ID lookup
// and calculation is a lot simpler.
func makeGlyphIndexFormat12(b binarySegm) (CMapGlyphIndex, error) {
	const headerSize = 16
	if headerSize > b.Size() {
		return nil, errFontFormat("cmap subtable bounds overflow")
	}
	size, _ := b.u32(4)
	grpCount, _ := b.u32(12)
	eLength := 12 * int(grpCount)
	if eLength > b.Size() || eLength+headerSize > int(size) {
		return nil, errFontFormat("cmap internal structure")
	}
	b = b[headerSize:size]
	// SequentialMapGroup Record:
	// Type     Name            Description
	// uint32   startCharCode   First character code in this group
	// uint32   endCharCode     Last character code in this group
	// uint32   startGlyphID    Glyph index corresponding to the starting character code
	groups := viewArray(b, 12) // 12 is byte size of group-record
	entries := make([]cmapEntry32, grpCount)
	for i := range entries {
		entries[i] = cmapEntry32{
			start: groups.Get(i).U32(0),
			end:   u32(groups.Get(i).Bytes()[4:]),
			delta: u32(groups.Get(i).Bytes()[8:]),
		}
	}
	return format12GlyphIndex{
		grpCnt:  int(grpCount),
		entries: entries,
	}, nil
}
