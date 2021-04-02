package ot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Code comment often will cite passage from the
// OpenType specification version 1.8.4;
// see https://docs.microsoft.com/en-us/typography/opentype/spec/.

// ---------------------------------------------------------------------------

// Parse parses an OpenType font from a byte slice.
// An ot.Font needs ongoing access to the fonts byte-data after the Parse function returns.
// Its elements are assumed immutable while the ot.Font remains in use.
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
	case T("BASE"):
		return parseBase(t, b, offset, size)
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

// --- BASE table ------------------------------------------------------------

// The Baseline table (BASE) provides information used to align glyphs of different
// scripts and sizes in a line of text, whether the glyphs are in the same font or
// in different fonts.
func parseBase(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	var err error
	base := newBaseTable(tag, b, offset, size)
	// The BASE table begins with offsets to Axis tables that describe layout data for
	// the horizontal and vertical layout directions of text. A font can provide layout
	// data for both text directions or for only one text direction.
	xaxis, errx := parseLink16(b, 4, b, "Axis")
	yaxis, erry := parseLink16(b, 6, b, "Axis")
	if errx != nil || erry != nil {
		return nil, errFontFormat("BASE table axis-tables")
	}
	err = parseBaseAxis(base, 0, xaxis, err)
	err = parseBaseAxis(base, 1, yaxis, err)
	if err != nil {
		trace().Errorf("error parsing BASE table: %v", err)
		return base, err
	}
	return base, err
}

// An Axis table consists of offsets, measured from the beginning of the Axis table,
// to a BaseTagList and a BaseScriptList.
// link may be NULL.
func parseBaseAxis(base *BaseTable, hOrV int, link NavLink, err error) error {
	if err != nil {
		return err
	}
	base.axisTables[hOrV] = AxisTable{}
	if link.IsNull() {
		return nil
	}
	axisHeader := link.Jump()
	axisbase := axisHeader.Bytes()
	// The BaseTagList enumerates all baselines used to render the scripts in the
	// text layout direction. If no baseline data is available for a text direction,
	// the offset to the corresponding BaseTagList may be set to NULL.
	if basetags, err := parseLink16(axisbase, 0, axisbase, "BaseTagList"); err == nil {
		b := basetags.Jump()
		base.axisTables[hOrV].baselineTags = parseTagList(b.Bytes())
		trace().Debugf("axis table %d has %d entries", hOrV,
			base.axisTables[hOrV].baselineTags.Count)
	}
	// For each script listed in the BaseScriptList table, a BaseScriptRecord must be
	// defined that identifies the script and references its layout data.
	// BaseScriptRecords are stored in the baseScriptRecords array, ordered
	// alphabetically by the baseScriptTag in each record.
	if basescripts, err := parseLink16(axisbase, 2, axisbase, "BaseScriptList"); err == nil {
		b := basescripts.Jump()
		base.axisTables[hOrV].baseScriptRecords = parseTagRecordMap16(b.Bytes(),
			0, b.Bytes(), "BaseScriptList", "BaseScript")

		trace().Debugf("axis table %d has %d entries", hOrV,
			base.axisTables[hOrV].baselineTags.Count)
	}
	trace().Infof("BASE axis %d has no/unreadable entires", hOrV)
	return nil
}

// --- CMap table ------------------------------------------------------------

// This table defines mapping of character codes to a default glyph index. Different
// subtables may be defined that each contain mappings for different character encoding
// schemes. The table header indicates the character encodings for which subtables are
// present.
//
// From the spec.: “Apart from a format 14 subtable, all other subtables are exclusive:
// applications should select and use one and ignore the others. […]
// If a font includes Unicode subtables for both 16-bit encoding (typically, format 4)
// and also 32-bit encoding (formats 10 or 12), then the characters supported by the
// subtable for 32-bit encoding should be a superset of the characters supported by
// the subtable for 16-bit encoding, and the 32-bit encoding should be used by
// applications. Fonts should not include 16-bit Unicode subtables using both format 4
// and format 6; format 4 should be used. Similarly, fonts should not include 32-bit
// Unicode subtables using both format 10 and format 12; format 12 should be used.
// If a font includes encoding records for Unicode subtables of the same format but
// with different platform IDs, an application may choose which to select, but should
// make this selection consistently each time the font is used.”
//
// From Apple: // https://developer.apple.com/fonts/TrueType-Reference-Manual/RM06/Chap6cmap.html
// “The use of the Macintosh platformID is currently discouraged. Subtables with a
//  Macintosh platformID are only required for backwards compatibility.”
// and:
// “The Unicode platform's platform-specific ID 6 was intended to mark a 'cmap' subtable
//  as one used by a last resort font. This is not required by any Apple platform.”
//
// All in all, we only support the following plaform/encoding/format combinations:
//   0 (Unicode)  3    4   Unicode BMB
//   0 (Unicode)  4    12  Unicode full
//   3 (Win)      1    4   Unicode BMP
//   3 (Win)      10   12  Unicode full
//
func parseCMap(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	n, _ := b.u16(2) // number of sub-tables
	trace().Debugf("font cmap has %d sub-tables in %d|%d bytes", n, len(b), size)
	t := newCMapTable(tag, b, offset, size)
	const headerSize, entrySize = 4, 8
	if size < headerSize+entrySize*uint32(n) {
		return nil, errFontFormat("size of cmap table")
	}
	var enc encodingRecord
	for i := 0; i < int(n); i++ {
		rec, _ := b.view(headerSize+entrySize*i, entrySize)
		pid, psid := u16(rec), u16(rec[2:])
		width := platformEncodingWidth(pid, psid)
		if width <= enc.width {
			continue
		}
		link, err := parseLink32(rec, 4, b, "cmap.Subtable")
		if err != nil {
			trace().Infof("cmap sub-table cannot be parsed")
			continue
		}
		subtable := link.Jump()
		format := subtable.U16(0)
		trace().Debugf("cmap table contains subtable with format %d", format)
		if supportedCmapFormat(format, pid, psid) {
			enc.width = width
			enc.format = format
			enc.link = link
		}
	}
	if enc.width == 0 {
		return nil, errFontFormat("no supported cmap format found")
	}
	var err error
	if t.GlyphIndexMap, err = makeGlyphIndex(b, enc); err != nil {
		return nil, err
	}
	return t, nil
}

type encodingRecord struct {
	platformId uint16
	encodingId uint16
	link       NavLink
	format     uint16
	size       int
	width      int // encoding width in bytes
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
// platform. In the real world, fonts usually have just one kern sub-table, and
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
			trace().Infof("kern sub-table size should be 0x%x, but given as 0x%x; fixing",
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

// The Glyph Definition (GDEF) table provides various glyph properties used in
// OpenType Layout processing.
func parseGDef(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	var err error
	gdef := newGDefTable(tag, b, offset, size)
	err = parseGDefHeader(gdef, b, err)
	err = parseGlyphClassDefinitions(gdef, b, err)
	err = parseAttachmentPointList(gdef, b, err)
	// we currently do not parse the Ligature Caret List Table
	err = parseMarkAttachmentClassDef(gdef, b, err)
	err = parseMarkGlyphSets(gdef, b, err)
	if err != nil {
		trace().Errorf("error parsing GDEF table: %v", err)
		return gdef, err
	}
	mj, mn := gdef.Header().Version()
	trace().Debugf("GDEF table has version %d.%d", mj, mn)
	return gdef, err
}

// The GDEF table begins with a header that starts with a version number. Three
// versions are defined. Version 1.0 contains an offset to a Glyph Class Definition
// table (GlyphClassDef), an offset to an Attachment List table (AttachList), an offset
// to a Ligature Caret List table (LigCaretList), and an offset to a Mark Attachment
// Class Definition table (MarkAttachClassDef). Version 1.2 also includes an offset to
// a Mark Glyph Sets Definition table (MarkGlyphSetsDef). Version 1.3 also includes an
// offset to an Item Variation Store table.
func parseGDefHeader(gdef *GDefTable, b fontBinSegm, err error) error {
	if err != nil {
		return err
	}
	h := GDefHeader{}
	r := bytes.NewReader(b)
	if err = binary.Read(r, binary.BigEndian, &h.gDefHeaderV1_0); err != nil {
		return err
	}
	headerlen := 12
	if h.versionHeader.Minor >= 2 {
		h.MarkGlyphSetsDefOffset, _ = b.u16(headerlen)
		headerlen += 2
	}
	if h.versionHeader.Minor >= 3 {
		h.ItemVarStoreOffset, _ = b.u32(headerlen)
		headerlen += 4
	}
	gdef.header = h
	gdef.header.headerSize = uint8(headerlen)
	return err
}

// This table uses the same format as the Class Definition table (defined in the
// OpenType Layout Common Table Formats chapter).
func parseGlyphClassDefinitions(gdef *GDefTable, b fontBinSegm, err error) error {
	if err != nil {
		return err
	}
	offset := gdef.Header().offsetFor(GDefGlyphClassDefSection)
	if offset >= len(b) {
		return io.ErrUnexpectedEOF
	}
	b = b[offset:]
	cdef, err := parseClassDefinitions(b)
	if err != nil {
		return err
	}
	gdef.GlyphClassDef = cdef
	return nil
}

/*
AttachList:
Type      Name                            Description
---------+-------------------------------+-----------------------
Offset16  coverageOffset                  Offset to Coverage table - from beginning of AttachList table
uint16    glyphCount                      Number of glyphs with attachment points
Offset16  attachPointOffsets[glyphCount]  Array of offsets to AttachPoint tables-from beginning of
                                          AttachList table-in Coverage Index order
*/
func parseAttachmentPointList(gdef *GDefTable, b fontBinSegm, err error) error {
	if err != nil {
		return err
	}
	offset := gdef.Header().offsetFor(GDefAttachListSection)
	if offset >= len(b) {
		return io.ErrUnexpectedEOF
	}
	b = b[offset:]
	if count, err := b.u16(2); count == 0 {
		if err != nil {
			return errFontFormat("GDEF has corrupt attachment point list")
		}
		return nil // no entries
	}
	covOffset := u16(b)
	coverage := parseCoverage(b[covOffset:])
	if coverage.GlyphRange == nil {
		return errFontFormat("GDEF attachement point coverage table unreadable")
	}
	count, _ := b.u16(2)
	gdef.AttachmentPointList = AttachmentPointList{
		Count:              int(count),
		Coverage:           coverage.GlyphRange,
		attachPointOffsets: b[4:],
	}
	return nil
}

// A Mark Attachment Class Definition Table defines the class to which a mark glyph may
// belong. This table uses the same format as the Class Definition table.
func parseMarkAttachmentClassDef(gdef *GDefTable, b fontBinSegm, err error) error {
	if err != nil {
		return err
	}
	offset := gdef.Header().offsetFor(GDefMarkAttachClassSection)
	if offset >= len(b) {
		return io.ErrUnexpectedEOF
	}
	b = b[offset:]
	cdef, err := parseClassDefinitions(b)
	if err != nil {
		return err
	}
	gdef.MarkAttachmentClassDef = cdef
	return nil
}

// Mark glyph sets are defined in a MarkGlyphSets table, which contains offsets to
// individual sets each represented by a standard Coverage table.
func parseMarkGlyphSets(gdef *GDefTable, b fontBinSegm, err error) error {
	if err != nil {
		return err
	}
	offset := gdef.Header().offsetFor(GDefMarkGlyphSetsDefSection)
	if offset >= len(b) {
		return io.ErrUnexpectedEOF
	}
	b = b[offset:]
	count, _ := b.u16(2)
	for i := 0; i < int(count); i++ {
		covOffset, _ := b.u32(i * 4)
		coverage := parseCoverage(b[covOffset:])
		if coverage.GlyphRange == nil {
			return errFontFormat("GDEF mark glyph set coverage table unreadable")
		}
		gdef.MarkGlyphSets = append(gdef.MarkGlyphSets, coverage.GlyphRange)
	}
	return nil
}

// --- GPOS table ------------------------------------------------------------

// The Glyph Positioning table (GPOS) provides precise control over glyph placement for
// sophisticated text layout and rendering in each script and language system that a font
// supports.
func parseGPos(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	var err error
	gpos := newGPosTable(tag, b, offset, size)
	err = parseLayoutHeader(&gpos.LayoutTable, b, err)
	err = parseLookupList(&gpos.LayoutTable, b, err)
	err = parseFeatureList(&gpos.LayoutTable, b, err)
	err = parseScriptList(&gpos.LayoutTable, b, err)
	if err != nil {
		trace().Errorf("error parsing GPOS table: %v", err)
		return gpos, err
	}
	mj, mn := gpos.header.Version()
	trace().Debugf("GPOS table has version %d.%d", mj, mn)
	trace().Debugf("GPOS table has %d lookup list entries", gpos.LookupList.length)
	return gpos, err
}

// --- GSUB table ------------------------------------------------------------

// The Glyph Substitution (GSUB) table provides data for substition of glyphs for
// appropriate rendering of scripts, such as cursively-connecting forms in Arabic script,
// or for advanced typographic effects, such as ligatures.
func parseGSub(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	var err error
	gsub := newGSubTable(tag, b, offset, size)
	err = parseLayoutHeader(&gsub.LayoutTable, b, err)
	err = parseLookupList(&gsub.LayoutTable, b, err)
	err = parseFeatureList(&gsub.LayoutTable, b, err)
	err = parseScriptList(&gsub.LayoutTable, b, err)
	if err != nil {
		trace().Errorf("error parsing GSUB table: %v", err)
		return gsub, err
	}
	mj, mn := gsub.header.Version()
	trace().Debugf("GSUB table has version %d.%d", mj, mn)
	trace().Debugf("GSUB table has %d lookup list entries", gsub.LookupList.length)
	return gsub, err
}

// --- Common code for GPos and GSub -----------------------------------------

// parseLayoutHeader parses a layout table header, i.e. reads version information
// and header information (containing offsets).
// Supports header versions 1.0 and 1.1
func parseLayoutHeader(lytt *LayoutTable, b fontBinSegm, err error) error {
	if err != nil {
		return err
	}
	h := &LayoutHeader{}
	r := bytes.NewReader(b)
	if err = binary.Read(r, binary.BigEndian, &h.versionHeader); err != nil {
		return err
	}
	if h.Major != 1 || (h.Minor != 0 && h.Minor != 1) {
		return fmt.Errorf("unsupported layout version (major: %d, minor: %d)",
			h.Major, h.Minor)
	}
	switch h.Minor {
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

// No longer used.
//
// parseLookup parses a single Lookup record. b expected to be the beginning of LookupList.
// See https://www.microsoft.com/typography/otspec/chapter2.htm#featTbl
//
// A lookup record starts with type and flag fields, followed by a count of
// sub-tables.
func parseLookup(b fontBinSegm, offset uint16) (*Lookup, error) {
	if int(offset) >= len(b) {
		return nil, io.ErrUnexpectedEOF
	}
	r := bytes.NewReader(b[offset:])
	var lookup Lookup
	if err := binary.Read(r, binary.BigEndian, &lookup.lookupInfo); err != nil {
		return nil, fmt.Errorf("reading lookupRecord: %s", err)
	}
	//trace().Debugf("lookup table (%d) has %d subtables", lookup.Type, lookup.SubRecordCount)
	subs := make([]uint16, lookup.SubTableCount, lookup.SubTableCount)
	if err := binary.Read(r, binary.BigEndian, &subs); err != nil {
		return nil, fmt.Errorf("reading lookupRecord: %s", err)
	}
	// lookup.subrecordOffsets = subs
	// if lookup.Type != GSUB_LUTYPE_Single {
	// 	return &lookup, nil
	// }
	// for i := 0; i < len(subs); i++ {
	// 	off := subs[i]
	// 	//trace().Debugf("offset of sub-table[%d] = %d", i, subs[i])
	// 	r = bytes.NewReader(b[offset+off:])
	// 	subst := lookupSubstFormat1{}
	// 	if err := binary.Read(r, binary.BigEndian, &subst); err != nil {
	// 		return nil, fmt.Errorf("reading lookupRecord: %s", err)
	// 	}
	// 	//trace().Debugf("   format spec = %d", subst.Format)
	// 	if subst.Format == 2 {
	// 		subst2 := lookupSubstFormat2{}
	// 		subst2.lookupSubstFormat1 = subst
	// 		if err := read16arr(r, &subst2.SubstituteGlyphIDs, int(subst.Glyphs)); err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// }
	// TODO Read lookup.MarkFilteringSet  ?
	return &lookup, nil
}

// parseLookupList parses the LookupList.
// See https://www.microsoft.com/typography/otspec/chapter2.htm#lulTbl
func parseLookupList(lytt *LayoutTable, b fontBinSegm, err error) error {
	if err != nil {
		return err
	}
	lloffset := lytt.header.offsetFor(LayoutLookupSection)
	if lloffset >= len(b) {
		return io.ErrUnexpectedEOF
	}
	b = b[lloffset:]
	//
	ll := LookupList{base: b}
	ll.array, ll.err = parseArray16(b, 0)
	lytt.LookupList = ll
	// r := bytes.NewReader(b)
	// var count uint16
	// if err := binary.Read(r, binary.BigEndian, &count); err != nil {
	// 	return fmt.Errorf("reading lookup count: %s", err)
	// }
	// trace().Debugf("font has %d lookup list entries", count)
	// if count > 0 {
	// 	lookupOffsets := make([]uint16, count, count)
	// 	if err := binary.Read(r, binary.BigEndian, &lookupOffsets); err != nil {
	// 		return fmt.Errorf("reading lookup offsets: %s", err)
	// 	}
	// 	lytt.LookupList = nil
	// 	for i := 0; i < int(count); i++ {
	// 		//trace().Debugf("lookup offset #%d = %d", i, lookupOffsets[i])
	// 		lookup, err := parseLookup(b, lookupOffsets[i])
	// 		if err != nil {
	// 			return err
	// 		}
	// 		lytt.LookupList = append(lytt.LookupList, lookup)
	// 	}
	// }
	return nil
}

func parseLookupSubtable(b fontBinSegm, lookupType LayoutTableLookupType) LookupSubtable {
	if len(b) < 4 {
		return LookupSubtable{}
	}
	if IsGPosLookupType(lookupType) {
		return parseGPosLookupSubtable(b, GPosLookupType(lookupType))
	}
	return parseGSubLookupSubtable(b, GSubLookupType(lookupType))
}

func parseGSubLookupSubtable(b fontBinSegm, lookupType LayoutTableLookupType) LookupSubtable {
	format := b.U16(0)
	trace().Debugf("parsing GSUB sub-table type %s, format %d", lookupType.GSubString(), format)
	sub := LookupSubtable{lookupType: lookupType, format: format}
	if lookupType != 7 { // GSUB type Extension has not coverage table
		sub.coverage = parseCoverage(b)
	}
	switch lookupType {
	case 1:
		return parseGSubLookupSubtableType1(b, sub)
	case 2, 3:
		return parseGSubLookupSubtableType2or3(b, sub)
	case 4:
		return parseGSubLookupSubtableType4(b, sub)
	case 7:
		return parseGSubLookupSubtableType7(b, sub)
	}
	return LookupSubtable{} // TODO
}

// LookupType 1: Single Substitution Subtable
// Single substitution (SingleSubst) subtables tell a client to replace a single glyph with
// another glyph.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-1-single-substitution-subtable
func parseGSubLookupSubtableType1(b fontBinSegm, sub LookupSubtable) LookupSubtable {
	if sub.format == 1 {
		sub.support = int16(b.U16(4))
	} else {
		sub.index = parseVarArrary16(b, 4, 1, "LookupSubtable")
	}
	return sub
}

// LookupType 2: Multiple Substitution Subtable
// A Multiple Substitution (MultipleSubst) subtable replaces a single glyph with more than
// one glyph, as when multiple glyphs replace a single ligature.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-2-multiple-substitution-subtable
// LookupType 3: Alternate Substitution Subtable
// An Alternate Substitution (AlternateSubst) subtable identifies any number of aesthetic
// alternatives from which a user can choose a glyph variant to replace the input glyph.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-3-alternate-substitution-subtable
func parseGSubLookupSubtableType2or3(b fontBinSegm, sub LookupSubtable) LookupSubtable {
	sub.index = parseVarArrary16(b, 4, 2, "LookupSubtable")
	return sub
}

// LookupType 4: Ligature Substitution Subtable
// A Ligature Substitution (LigatureSubst) subtable identifies ligature substitutions where
// a single glyph replaces multiple glyphs. One LigatureSubst subtable can specify any
// number of ligature substitutions.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-4-ligature-substitution-subtable
func parseGSubLookupSubtableType4(b fontBinSegm, sub LookupSubtable) LookupSubtable {
	sub.index = parseVarArrary16(b, 4, 2, "LookupSubtable")
	return sub
}

// LookupType 7: Extension Substitution
// This lookup provides a mechanism whereby any other lookup type’s subtables are stored at
// a 32-bit offset location in the GSUB table.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-7-extension-substitution
func parseGSubLookupSubtableType7(b fontBinSegm, sub LookupSubtable) LookupSubtable {
	if b.Size() < 8 {
		trace().Errorf("OpenType GSUB lookup subtable type %d corrupt", sub.lookupType)
		return LookupSubtable{}
	}
	if sub.lookupType = LayoutTableLookupType(b.U16(2)); sub.lookupType == GSubLookupTypeExtensionSubs {
		trace().Errorf("OpenType GSUB lookup subtable type 7 recursion detected")
		return LookupSubtable{}
	}
	trace().Debugf("OpenType GSUB extension subtable is of type %s", sub.lookupType.GSubString())
	link, _ := parseLink32(b, 4, b, "ext.LookupSubtable")
	loc := link.Jump()
	return parseGSubLookupSubtable(loc.Bytes(), sub.lookupType)
}

func parseGPosLookupSubtable(b fontBinSegm, lookupType LayoutTableLookupType) LookupSubtable {
	format := b.U16(0)
	trace().Debugf("parsing GPOS sub-table type %s, format %d", lookupType.GPosString(), format)
	panic("TODO GPOS Lookup Subtable")
	//return LookupSubtable{}
}

// --- Feature list ----------------------------------------------------------

// The FeatureList table enumerates features in an array of records (FeatureRecord) and
// specifies the total number of features (FeatureCount). Every feature must have a
// FeatureRecord, which consists of a FeatureTag that identifies the feature and an offset
// to a Feature table (described next). The FeatureRecord array is arranged alphabetically
// by FeatureTag names.
func parseFeatureList(lytt *LayoutTable, b []byte, err error) error {
	if err != nil {
		return err
	}
	// floffset := lytt.header.OffsetFor(LayoutFeatureSection)
	// if floffset >= len(b) {
	// 	return io.ErrUnexpectedEOF
	// }
	// flist, n, err := lytt.data.varLenView(floffset, 2, 0, 6)
	// r := bytes.NewReader(flist[2:])
	// frecords := make([]featureRecord, n, n)
	// if err := binary.Read(r, binary.BigEndian, &frecords); err != nil {
	// 	return fmt.Errorf("reading feature records: %s", err)
	// }
	// lytt.features = frecords
	// return nil
	lytt.FeatureList = tagRecordMap16{}
	link := link16{base: b, offset: uint16(lytt.header.offsetFor(LayoutFeatureSection))}
	features := link.Jump() // now we stand at the FeatureList table
	featureRecords := parseTagRecordMap16(features.Bytes(), 0, features.Bytes(), "FeatureList", "Feature")
	lytt.FeatureList = featureRecords
	return nil
}

// b+offset has to be positioned at the start of the feature index list block, e.g.,
// the second uint16 of a LangSys table:
// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#language-system-table
//
// uint16  requiredFeatureIndex               Index of a feature required for this language system
// uint16  featureIndexCount                  Number of feature index values for this language system
// uint16  featureIndices[featureIndexCount]  Array of indices into the FeatureList, in arbitrary order
//
func parseLangSys(b fontBinSegm, offset int, target string) (langSys, error) {
	lsys := langSys{}
	if len(b) < offset+4 {
		return lsys, errBufferBounds
	}
	trace().Debugf("parsing LangSys (%s)", target)
	b = b[offset:]
	lsys.mandatory, _ = b.u16(0)
	features, err := parseArray16(b, 2)
	if err != nil {
		return lsys, err
	}
	lsys.featureIndices = features
	return lsys, nil
}

// --- Script list -----------------------------------------------------------

// A ScriptList table consists of a count of the scripts represented by the glyphs in the
// font (ScriptCount) and an array of records (ScriptRecord), one for each script for which
// the font defines script-specific features (a script without script-specific features
// does not need a ScriptRecord). Each ScriptRecord consists of a ScriptTag that identifies
// a script, and an offset to a Script table. The ScriptRecord array is stored in
// alphabetic order of the script tags.
func parseScriptList(lytt *LayoutTable, b fontBinSegm, err error) error {
	if err != nil {
		return err
	}
	lytt.ScriptList = tagRecordMap16{}
	link := link16{base: b, offset: uint16(lytt.header.offsetFor(LayoutScriptSection))}
	scripts := link.Jump() // now we stand at the ScriptList table
	scriptRecords := parseTagRecordMap16(scripts.Bytes(), 0, scripts.Bytes(), "ScriptList", "Script")
	lytt.ScriptList = scriptRecords
	return nil
}

// --- parse class def table -------------------------------------------------

// The ClassDef table can have either of two formats: one that assigns a range of
// consecutive glyph indices to different classes, or one that puts groups of consecutive
// glyph indices into the same class.
func parseClassDefinitions(b fontBinSegm) (ClassDefinitions, error) {
	cdef := ClassDefinitions{}
	r := bytes.NewReader(b)
	if err := binary.Read(r, binary.BigEndian, &cdef.format); err != nil {
		return cdef, err
	}
	var n, g uint16
	if cdef.format == 1 {
		n, _ = b.u16(4) // number of glyph IDs in table
		g, _ = b.u16(2) // start glyph ID
	} else if cdef.format == 2 {
		n, _ = b.u16(2) // number of glyph ID ranges in table
	} else {
		return cdef, errFontFormat(fmt.Sprintf("unknown ClassDef format %d", n))
	}
	records := cdef.makeArray(b, int(n), cdef.format)
	cdef.setRecords(records, GlyphIndex(g))
	return cdef, nil
}

// --- parse coverage table-module -------------------------------------------

// Read a coverage table-module, which comes in two formats (1 and 2).
// A Coverage table defines a unique index value, the Coverage Index, for each
// covered glyph.
func parseCoverage(b fontBinSegm) Coverage {
	h := coverageHeader{}
	r := bytes.NewReader(b)
	if err := binary.Read(r, binary.BigEndian, &h); err != nil {
		return Coverage{}
	}
	return Coverage{
		coverageHeader: h,
		GlyphRange:     buildGlyphRangeFromCoverage(h, b),
	}
}

// --- Sequence context ------------------------------------------------------

// The contextual lookup types support specifying input glyph sequences that will can be
// acted upon, as well as a list of actions to be taken on any glyph within the sequence.
// Actions are specified as references to separate nested lookups (an index into the
// LookupList). The actions are specified for each glyph position, but the entire sequence
// must be matched, and so the actions are specified in a context-sensitive manner.

// Three subtable formats are defined, which describe the input sequences in different ways.
func parseSequenceContext(b fontBinSegm) (sequenceContext, error) {
	if len(b) <= 2 {
		return sequenceContext{}, errFontFormat("corrupt sequence context")
	}
	format := b.U16(0)
	switch format {
	case 1:
		parseSequenceContextFormat1(format, b)
	case 2:
		parseSequenceContextFormat2(format, b)
	case 3:
		parseSequenceContextFormat3(format, b)
	}
	return sequenceContext{}, errFontFormat(
		fmt.Sprintf("unknown sequence context format %d", format))
}

// SequenceContextFormat1: simple glyph contexts
// Type 	Name 	Description
// uint16 	format 	Format identifier: format = 1
// Offset16 	coverageOffset 	Offset to Coverage table, from beginning of SequenceContextFormat1 table
// uint16 	seqRuleSetCount 	Number of SequenceRuleSet tables
// Offset16 	seqRuleSetOffsets[seqRuleSetCount] 	Array of offsets to SequenceRuleSet tables, from beginning of SequenceContextFormat1 table (offsets may be NULL)
func parseSequenceContextFormat1(fmtno uint16, b fontBinSegm) (sequenceContext, error) {
	if len(b) <= 6 {
		return sequenceContext{}, errFontFormat("corrupt sequence context")
	}
	seqctx := sequenceContext{format: fmtno, coverage: make([]Coverage, 1)}
	link, err := parseLink16(b, 2, b, "SequenceContext Coverage")
	if err != nil {
		return sequenceContext{}, errFontFormat("corrupt sequence context")
	}
	cov := link.Jump()
	seqctx.coverage[0] = parseCoverage(cov.Bytes())
	seqctx.rules = parseVarArrary16(b, 4, 2, "SequenceContext")
	return seqctx, nil
}

// SequenceContextFormat2 table:
// Type 	Name 	Description
// uint16 	format 	Format identifier: format = 2
// Offset16 	coverageOffset 	Offset to Coverage table, from beginning of SequenceContextFormat2 table
// Offset16 	classDefOffset 	Offset to ClassDef table, from beginning of SequenceContextFormat2 table
// uint16 	classSeqRuleSetCount 	Number of ClassSequenceRuleSet tables
// Offset16 	classSeqRuleSetOffsets[classSeqRuleSetCount] 	Array of offsets to ClassSequenceRuleSet tables, from beginning of SequenceContextFormat2 table (may be NULL)
func parseSequenceContextFormat2(fmtno uint16, b fontBinSegm) (sequenceContext, error) {
	if len(b) <= 8 {
		return sequenceContext{}, errFontFormat("corrupt sequence context")
	}
	seqctx := sequenceContext{format: fmtno, coverage: make([]Coverage, 1)}
	link, err := parseLink16(b, 2, b, "SequenceContext Coverage")
	if err != nil {
		return sequenceContext{}, errFontFormat("corrupt sequence context")
	}
	cov := link.Jump()
	seqctx.coverage[0] = parseCoverage(cov.Bytes())
	link, err = parseLink16(b, 2, b, "SequenceContext Coverage")
	if err != nil {
		return sequenceContext{}, errFontFormat("corrupt sequence context")
	}
	classdef := link.Jump()
	seqctx.classDef, err = parseClassDefinitions(classdef.Bytes())
	seqctx.rules = parseVarArrary16(b, 6, 2, "SequenceContext")
	return seqctx, nil
}

// The SequenceContextFormat3 table specifies exactly one input sequence pattern. It has an
// array of offsets to coverage tables. These correspond, in order, to the positions in the
// input sequence pattern.
//
// SequenceContextFormat3 table:
// Type 	Name 	Description
// uint16 	format 	Format identifier: format = 3
// uint16 	glyphCount 	Number of glyphs in the input sequence
// uint16 	seqLookupCount 	Number of SequenceLookupRecords
// Offset16 	coverageOffsets[glyphCount] 	Array of offsets to Coverage tables, from beginning of SequenceContextFormat3 subtable
// SequenceLookupRecord 	seqLookupRecords[seqLookupCount] 	Array of SequenceLookupRecords
func parseSequenceContextFormat3(fmtno uint16, b fontBinSegm) (sequenceContext, error) {
	if len(b) <= 8 {
		return sequenceContext{}, errFontFormat("corrupt sequence context")
	}
	count := b.U16(2)
	seqctx := sequenceContext{format: fmtno}
	seqctx.coverage = make([]Coverage, count)
	for i := 0; i < int(count); i++ {
		link, err := parseLink16(b, 6+i*2, b, "SequenceContext Coverage")
		if err != nil {
			return sequenceContext{}, errFontFormat("corrupt sequence context")
		}
		cov := link.Jump()
		seqctx.coverage[i] = parseCoverage(cov.Bytes())
	}
	count = b.U16(6 + int(count)*2)
	seqctx.lookupRecords = array{
		recordSize: 4, // 2* sizeof(uint16)
		length:     int(count),
		loc:        b[4+count*2:],
	}
	return seqctx, nil
}

// ClassSequenceRule table:
// Type     Name                          Description
// uint16   glyphCount                    Number of glyphs to be matched
// uint16   seqLookupCount                Number of SequenceLookupRecords
// uint16   inputSequence[glyphCount-1]   Sequence of classes to be matched to the input glyph sequence, beginning with the second glyph position
// SequenceLookupRecord seqLookupRecords[seqLookupCount]   Array of SequenceLookupRecords
func parseSequenceRule(b fontBinSegm) sequenceRule {
	seqrule := sequenceRule{}
	seqrule.glyphCount = b.U16(0)
	seqrule.inputSequence = array{
		recordSize: 2, // sizeof(uint16)
		length:     int(seqrule.glyphCount) - 1,
	}
	seqrule.inputSequence.loc = b[4 : 4+seqrule.inputSequence.length*2]
	// SequenceLookupRecord:
	// Type     Name             Description
	// uint16   sequenceIndex    Index (zero-based) into the input glyph sequence
	// uint16   lookupListIndex  Index (zero-based) into the LookupList
	cnt := b.U16(2)
	seqrule.lookupRecords = array{
		recordSize: 4, // 2* sizeof(uint16)
		length:     int(cnt),
		loc:        b[4+seqrule.inputSequence.length*2:],
	}
	return seqrule
}

// --- Names -----------------------------------------------------------------

func parseNames(b fontBinSegm) (nameNames, error) {
	if len(b) < 6 {
		return nameNames{}, errFontFormat("name section corrupt")
	}
	N, _ := b.u16(2)
	names := nameNames{}
	strOffset, _ := b.u16(4)
	names.strbuf = b[strOffset:]
	trace().Debugf("name table has %d strings, starting at %d", N, strOffset)
	if len(b) < 6+12*int(N) {
		return nameNames{}, errFontFormat("name section corrupt")
	}
	recs := b[6 : 6+12*int(N)]
	names.nameRecs = viewArray(recs, 12)
	return names, nil
}
