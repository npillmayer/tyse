package ot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Parse parses an OpenType font from a byte slice.
func Parse(font []byte) (*Font, error) {
	// https://www.microsoft.com/typography/otspec/otff.htm: Offset Table is 12 bytes.
	r := bytes.NewReader(font)
	h := FontHeader{}
	if err := binary.Read(r, binary.BigEndian, &h); err != nil {
		return nil, err
	}
	trace().Debugf("header = %v, tag = %x|%s", h, h.FontType, Tag(h.FontType).String())
	if !(h.FontType == 0x4f54544f || // OTTO
		h.FontType == 0x00010000 || // TrueType
		h.FontType == 0x74727565) { // true
		return nil, errFontFormat(fmt.Sprintf("font type not supported: %x", h.FontType))
	}
	otf := &Font{Header: &h, tables: make(map[Tag]Table)}
	src := fontBinSegm(font)
	// "The Offset Table is followed immediately by the Table Record entries …
	// sorted in ascending order by tag", 16 bytes each.
	buf, err := src.view(12, 16*int(h.TableCount))
	if err != nil {
		return nil, errFontFormat("table record entries")
	}
	for b, prevTag := buf, Tag(0); len(b) > 0; b = b[16:] {
		tag := MakeTag(b)
		if tag < prevTag {
			return nil, errFontFormat("table order")
		}
		prevTag = tag
		off, size := u32(b[8:12]), u32(b[12:16])
		if off&3 != 0 { // ignore checksums, but "all tables must begin on four byte boundries".
			return nil, errFontFormat("invalid table offset")
		}
		otf.tables[tag], err = parseTable(tag, src[off:off+size], off, size)
		if err != nil {
			return nil, err
		}
	}
	// TODO consistency check
	//
	// The number of glyphs in the font is restricted only by the value stated in the 'head' table. The order in which glyphs are placed in a font is arbitrary.
	// Note that a font must have at least two glyphs, and that glyph index 0 musthave an outline. See Glyph Mappings for details.
	//
	return otf, nil
}

func parseTable(t Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	switch t {
	case T("cmap"):
		return parseCMap(t, b, offset, size)
	case T("head"):
		return parseHead(t, b, offset, size)
	case T("glyf"):
		return newTable(t, b, offset, size), nil // TODO
	case T("GDEF"):
		return parseGDef(t, b, offset, size)
	case T("GPOS"):
		return parseGPos(t, b, offset, size)
	case T("GSUB"):
		return parseGSub(t, b, offset, size)
	case T("hhea"):
		return parseHHea(t, b, offset, size)
	case T("hmtx"):
		return parseHMtx(t, b, offset, size)
	case T("kern"):
		return parseKern(t, b, offset, size)
	case T("loca"):
		return parseLoca(t, b, offset, size)
	case T("maxp"):
		return parseMaxP(t, b, offset, size)
	}
	trace().Infof("font contains table (%s), will not be interpreted", t)
	return newTable(t, b, offset, size), nil
}

// --- Head table ------------------------------------------------------------

func parseHead(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	if size < 54 {
		return nil, errFontFormat("size of head table")
	}
	t := newHeadTable(tag, b, offset, size)
	t.Flags, _ = b.u16(16)      // flags
	t.UnitsPerEm, _ = b.u16(18) // units per em
	// IndexToLocFormat is needed to interpret the loca table:
	// 0 for short offsets, 1 for long
	t.IndexToLocFormat, _ = b.u16(50)
	return t, nil
}

// --- CMap table ------------------------------------------------------------

func parseCMap(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	n, _ := b.u16(2)
	t := newCMapTable(tag, b, offset, size)
	t.numTables = int(n)
	const headerSize, entrySize = 4, 8
	if size < headerSize+entrySize*uint32(t.numTables) {
		return nil, errFontFormat("size of cmap table")
	}
	// Apart from a format 14 subtable, all other subtables are exclusive: applications
	// should select and use one and ignore the others. If a Unicode subtable is used
	// (platform 0, or platform 3 / encoding 1 or 10), then a format 14 subtable using
	// platform 0/encoding 5 can also be supplemented for mapping Unicode Variation Sequences.
	// If a font includes Unicode subtables for both 16-bit encoding (typically, format 4)
	// and also 32-bit encoding (formats 10 or 12), then the characters supported by the
	// subtable for 32-bit encoding should be a superset of the characters supported by
	// the subtable for 16-bit encoding, and the 32-bit encoding should be used by
	// applications. Fonts should not include 16-bit Unicode subtables using both format 4
	// and format 6; format 4 should be used. Similarly, fonts should not include 32-bit
	// Unicode subtables using both format 10 and format 12; format 12 should be used.
	// If a font includes encoding records for Unicode subtables of the same format but
	// with different platform IDs, an application may choose which to select, but should
	// make this selection consistently each time the font is used.
	var enc encodingRecord
	for i := 0; i < t.numTables; i++ {
		rec, _ := b.view(headerSize+entrySize*i, entrySize)
		pid, psid := u16(rec), u16(rec[2:])
		width := platformEncodingWidth(pid, psid)
		if width <= enc.width {
			continue
		}
		enc.offset = u32(rec[4:])
	}
	return t, nil
}

// --- Kern table ------------------------------------------------------------

type kernSubTableHeader struct {
	directory [4]uint16 // information to support binary search on sub-table
	offset    uint16    // start position of this sub-table's kern pairs
	length    uint32    // size of the sub-table in bytes, without header
	coverage  uint16    // info about type of information contained in this sub-table
}

// TrueType and OpenType slightly differ on formats of kern tables:
// see https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6kern.html
// and https://docs.microsoft.com/en-us/typography/opentype/spec/kern

// parseKern parses the kern table. There is significant confusion with this table
// concerning format differences between OpenType, TrueType, and fonts in the wild.
// We currently only support kern table format 0, which should be supported on any
// platform. In the real world, fonts usually have just on kern sub-table, and
// older Windows versions cannot handle more than one.
func parseKern(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	if size <= 4 {
		return nil, nil
	}
	var N, suboffset, subheaderlen int
	if version := u32(b); version == 0x00010000 {
		trace().Debugf("font has Apple TTF kern table format")
		n, _ := b.u32(4) // number of kerning tables is uint32
		N, suboffset, subheaderlen = int(n), 8, 16
	} else {
		trace().Debugf("font has OTF (MS) kern table format")
		n, _ := b.u16(2) // number of kerning tables is uint16
		N, suboffset, subheaderlen = int(n), 4, 14
	}
	trace().Debugf("kern table has %d sub-tables", N)
	t := newKernTable(tag, b, offset, size)
	for i := 0; i < N; i++ { // read in N sub-tables
		if suboffset+subheaderlen >= int(size) { // check for sub-table header size
			return nil, errFontFormat("kern table format")
		}
		h := kernSubTableHeader{
			offset: uint16(suboffset + subheaderlen),
			// sub-tables are of varying size; size may be off ⇒ see below
			length:   uint32(u16(b[suboffset+2:]) - uint16(subheaderlen)),
			coverage: u16(b[suboffset+4:]),
		}
		if format := h.coverage >> 8; format != 0 {
			trace().Infof("kern sub-table format %d not supported, ignoring sub-table", format)
			continue // we only support format 0 kerning tables; skip this one
		}
		h.directory = [4]uint16{
			u16(b[suboffset+subheaderlen-8:]),
			u16(b[suboffset+subheaderlen-6:]),
			u16(b[suboffset+subheaderlen-4:]),
			u16(b[suboffset+subheaderlen-2:]),
		}
		kerncnt := uint32(h.directory[0])
		trace().Debugf("kern sub-table has %d entries", kerncnt)
		// For some fonts, size calculation of kern sub-tables is off; see
		// https://github.com/fonttools/fonttools/issues/314#issuecomment-118116527
		// Testable with the Calibri font.
		sz := kerncnt * 6 // kern pair is of size 6
		if sz != h.length {
			trace().Infof("kern sub-table size should be %d, but given as %d; fixing",
				sz, h.length)
		}
		if uint32(suboffset)+sz >= size {
			return nil, errFontFormat("kern sub-table size exceeds kern table bounds")
		}
		t.headers = append(t.headers, h)
		suboffset += int(subheaderlen + int(h.length))
	}
	trace().Debugf("table kern has %d sub-table(s)", len(t.headers))
	return t, nil
}

// --- Loca table ------------------------------------------------------------

// Dependencies (taken from Apple Developer page about TrueType):
// The size of entries in the 'loca' table must be appropriate for the value of the
// indexToLocFormat field of the 'head' table. The number of entries must be the same
// as the numGlyphs field of the 'maxp' table.
// The 'loca' table is most intimately dependent upon the contents of the 'glyf' table
// and vice versa. Changes to the 'loca' table must not be made unless appropriate
// changes to the 'glyf' table are simultaneously made.
func parseLoca(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	return newLocaTable(tag, b, offset, size), nil
}

// --- MaxP table ------------------------------------------------------------

// This table establishes the memory requirements for this font. Fonts with CFF data
// must use Version 0.5 of this table, specifying only the numGlyphs field. Fonts
// with TrueType outlines must use Version 1.0 of this table, where all data is required.
func parseMaxP(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	if size <= 6 {
		return nil, nil
	}
	t := newMaxPTable(tag, b, offset, size)
	n, _ := b.u16(4)
	t.NumGlyphs = int(n)
	return t, nil
}

// --- HHea table ------------------------------------------------------------

// This table establishes the memory requirements for this font. Fonts with CFF data
// must use Version 0.5 of this table, specifying only the numGlyphs field. Fonts
// with TrueType outlines must use Version 1.0 of this table, where all data is required.
func parseHHea(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	if size == 0 {
		return nil, nil
	}
	trace().Debugf("HHea table has size %d", size)
	if size < 36 {
		return nil, errFontFormat("hhea table incomplete")
	}
	t := newHHeaTable(tag, b, offset, size)
	n, _ := b.u16(34)
	t.NumberOfHMetrics = int(n)
	return t, nil
}

// --- HMtx table ------------------------------------------------------------

// Dependencies (taken from Apple Developer page about TrueType):
// The value of the numOfLongHorMetrics field is found in the 'hhea' (Horizontal Header)
// table. Fonts that lack an 'hhea' table must not have an 'hmtx' table.
// Other tables may have information duplicating data contained in the 'hmtx' table.
// For example, glyph metrics can also be found in the 'hdmx' (Horizontal Device Metrics)
// table and 'bloc' (Bitmap Location) table. There is naturally no requirement that
// the ideal metrics of the 'hmtx' table be perfectly consistent with the device metrics
// found in other tables, but care should be taken that they are not significantly
// inconsistent.
func parseHMtx(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	if size == 0 {
		return nil, nil
	}
	t := newHMtxTable(tag, b, offset, size)
	return t, nil
}

// --- GDEF table ------------------------------------------------------------

func parseGDef(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	var err error
	gdef := newGDefTable(tag, b, offset, size)
	b, err = parseGDefHeader(gdef, b, err)
	b, err = parseGlyphClassDefinitions(gdef, b, err)
	//b, err = parse...(gdef, b, err)
	if err != nil {
		trace().Errorf("error parsing GDEF table: %v", err)
		return gdef, err
	}
	mj, mn := gdef.header.Version()
	trace().Debugf("GDEF table has version %d.%d", mj, mn)
	return gdef, err
}

func parseGDefHeader(gdef *GDefTable, b fontBinSegm, err error) (fontBinSegm, error) {
	if err != nil {
		return b, err
	}
	h := GDefHeader{}
	r := bytes.NewReader(b)
	if err = binary.Read(r, binary.BigEndian, &h.gDefHeaderV1_0); err != nil {
		return b, err
	}
	headerlen := 12
	if h.versionHeader.Minor >= 2 {
		h.MarkGlyphSetsDefOffset, _ = b.u16(headerlen)
		headerlen += 2
	}
	if h.versionHeader.Minor >= 3 {
		h.ItemVarStoreOffset, _ = b.u32(headerlen)
		headerlen += 2
	}
	gdef.header = h
	return b[headerlen:], err
}

func parseGlyphClassDefinitions(gdef *GDefTable, b fontBinSegm, err error) (fontBinSegm, error) {
	if err != nil {
		return b, err
	}
	cdef, err := parseClassDefinitions(b)
	if err != nil {
		return b, err
	}
	gdef.classDef = cdef
	return b[cdef.size:], nil
}

// --- GPOS table ------------------------------------------------------------

func parseGPos(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	var err error
	gpos := newGPosTable(tag, b, offset, size)
	err = parseLayoutHeader(gpos.LayoutBase(), b, err)
	err = parseLookupList(gpos.LayoutBase(), b, err)
	err = parseFeatureList(gpos.LayoutBase(), b, err)
	err = parseScriptList(gpos.LayoutBase(), b, err)
	if err != nil {
		trace().Errorf("error parsing GPOS table: %v", err)
		return gpos, err
	}
	mj, mn := gpos.header.Version()
	trace().Debugf("GPOS table has version %d.%d", mj, mn)
	trace().Debugf("GPOS table has %d lookup list entries", len(gpos.lookups))
	return gpos, err
}

// --- GSUB table ------------------------------------------------------------

func parseGSub(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	var err error
	gsub := newGSubTable(tag, b, offset, size)
	err = parseLayoutHeader(gsub.LayoutBase(), b, err)
	err = parseLookupList(gsub.LayoutBase(), b, err)
	err = parseFeatureList(gsub.LayoutBase(), b, err)
	err = parseScriptList(gsub.LayoutBase(), b, err)
	if err != nil {
		trace().Errorf("error parsing GSUB table: %v", err)
		return gsub, err
	}
	mj, mn := gsub.header.Version()
	trace().Debugf("GSUB table has version %d.%d", mj, mn)
	trace().Debugf("GSUB table has %d lookup list entries", len(gsub.lookups))
	return gsub, err
}

// --- Common code for GPos and GSub -----------------------------------------

// parseLayoutHeader parses a layout table header, i.e. reads version information
// and header information (containing offsets).
// Supports header versions 1.0 and 1.1
func parseLayoutHeader(lytt *LayoutTable, b []byte, err error) error {
	if err != nil {
		return err
	}
	h := &LayoutHeader{}
	r := bytes.NewReader(b)
	if err = binary.Read(r, binary.BigEndian, &h.version); err != nil {
		return err
	}
	if h.version.Major != 1 || (h.version.Minor != 0 && h.version.Minor != 1) {
		return fmt.Errorf("unsupported layout version (major: %d, minor: %d)",
			h.version.Major, h.version.Minor)
	}
	switch h.version.Minor {
	case 0:
		if err = binary.Read(r, binary.BigEndian, &h.offsets.layoutHeader10); err != nil {
			return err
		}
	case 1:
		if err = binary.Read(r, binary.BigEndian, &h.offsets); err != nil {
			return err
		}
	}
	lytt.header = h
	return nil
}

// --- Layout table lookup list ----------------------------------------------

// parseLookup parses a single Lookup record. b expected to be the beginning of LookupList.
// See https://www.microsoft.com/typography/otspec/chapter2.htm#featTbl
//
// A lookup record starts with type and flag fields, followed by a count of
// sub-tables.
func parseLookup(b []byte, offset uint16) (*lookupRecord, error) {
	if int(offset) >= len(b) {
		return nil, io.ErrUnexpectedEOF
	}
	r := bytes.NewReader(b[offset:])
	var lookup lookupRecord
	if err := binary.Read(r, binary.BigEndian, &lookup.lookupRecordInfo); err != nil {
		return nil, fmt.Errorf("reading lookupRecord: %s", err)
	}
	//trace().Debugf("lookup table (%d) has %d subtables", lookup.Type, lookup.SubRecordCount)
	subs := make([]uint16, lookup.SubRecordCount, lookup.SubRecordCount)
	if err := binary.Read(r, binary.BigEndian, &subs); err != nil {
		return nil, fmt.Errorf("reading lookupRecord: %s", err)
	}
	lookup.subrecordOffsets = subs
	if lookup.Type != GSUB_LUTYPE_Single {
		return &lookup, nil
	}
	for i := 0; i < len(subs); i++ {
		off := subs[i]
		//trace().Debugf("offset of sub-table[%d] = %d", i, subs[i])
		r = bytes.NewReader(b[offset+off:])
		subst := lookupSubstFormat1{}
		if err := binary.Read(r, binary.BigEndian, &subst); err != nil {
			return nil, fmt.Errorf("reading lookupRecord: %s", err)
		}
		//trace().Debugf("   format spec = %d", subst.Format)
		if subst.Format == 2 {
			subst2 := lookupSubstFormat2{}
			subst2.lookupSubstFormat1 = subst
			if err := read16arr(r, &subst2.SubstituteGlyphIDs, int(subst.Glyphs)); err != nil {
				return nil, err
			}
		}
	}
	// TODO Read lookup.MarkFilteringSet  ?
	return &lookup, nil
}

// parseLookupList parses the LookupList.
// See https://www.microsoft.com/typography/otspec/chapter2.htm#lulTbl
func parseLookupList(lytt *LayoutTable, b []byte, err error) error {
	if err != nil {
		return err
	}
	lloffset := lytt.header.Offset(LayoutLookupSection)
	if lloffset >= len(b) {
		return io.ErrUnexpectedEOF
	}
	b = b[lloffset:]
	r := bytes.NewReader(b)
	var count uint16
	if err := binary.Read(r, binary.BigEndian, &count); err != nil {
		return fmt.Errorf("reading lookup count: %s", err)
	}
	trace().Debugf("font has %d lookup list entries", count)
	if count > 0 {
		lookupOffsets := make([]uint16, count, count)
		if err := binary.Read(r, binary.BigEndian, &lookupOffsets); err != nil {
			return fmt.Errorf("reading lookup offsets: %s", err)
		}
		lytt.lookups = nil
		for i := 0; i < int(count); i++ {
			//trace().Debugf("lookup offset #%d = %d", i, lookupOffsets[i])
			lookup, err := parseLookup(b, lookupOffsets[i])
			if err != nil {
				return err
			}
			lytt.lookups = append(lytt.lookups, lookup)
		}
	}
	return nil
}

// --- Feature list ----------------------------------------------------------

func parseFeatureList(lytt *LayoutTable, b []byte, err error) error {
	if err != nil {
		return err
	}
	floffset := lytt.header.Offset(LayoutFeatureSection)
	if floffset >= len(b) {
		return io.ErrUnexpectedEOF
	}
	flist, n, err := lytt.data.varLenView(floffset, 2, 0, 6)
	r := bytes.NewReader(flist[2:])
	frecords := make([]featureRecord, n, n)
	if err := binary.Read(r, binary.BigEndian, &frecords); err != nil {
		return fmt.Errorf("reading feature records: %s", err)
	}
	lytt.features = frecords
	return nil
}

// --- Script list -----------------------------------------------------------

func parseScriptList(lytt *LayoutTable, b []byte, err error) error {
	if err != nil {
		return err
	}
	sloffset := lytt.header.Offset(LayoutScriptSection)
	if sloffset >= len(b) {
		return io.ErrUnexpectedEOF
	}
	slist, n, err := lytt.data.varLenView(sloffset, 2, 0, 6)
	r := bytes.NewReader(slist[2:])
	srecords := make([]scriptRecord, n, n)
	if err := binary.Read(r, binary.BigEndian, &srecords); err != nil {
		return fmt.Errorf("reading script records: %s", err)
	}
	lytt.scripts = srecords
	return nil
}

// --- parse class def table -------------------------------------------------

func parseClassDefinitions(b fontBinSegm) (ClassDefTable, error) {
	cdef := ClassDefTable{}
	r := bytes.NewReader(b)
	if err := binary.Read(r, binary.BigEndian, &cdef.format); err != nil {
		return cdef, err
	}
	var n uint16
	if cdef.format == 1 {
		n, _ = b.u16(4)
	} else if cdef.format == 2 {
		n, _ = b.u16(2)
	} else {
		return cdef, errFontFormat(fmt.Sprintf("unknown ClassDef format %d", n))
	}
	cdef.count = int(n)
	cdef.size = cdef.calcSize(int(n))
	return cdef, nil
}

// ---------------------------------------------------------------------------

func read16arr(r *bytes.Reader, arr *[]uint16, size int) error {
	*arr = make([]uint16, size, size)
	//r := bytes.NewReader(b[offset:])
	return binary.Read(r, binary.BigEndian, arr)
}
