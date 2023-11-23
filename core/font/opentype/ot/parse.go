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
	tracer().Debugf("header = %v, tag = %x|%s", h, h.FontType, Tag(h.FontType).String())
	if !(h.FontType == 0x4f54544f || // OTTO
		h.FontType == 0x00010000 || // TrueType
		h.FontType == 0x74727565) { // true
		return nil, errFontFormat(fmt.Sprintf("font type not supported: %x", h.FontType))
	}
	otf := &Font{Header: &h, tables: make(map[Tag]Table)}
	src := binarySegm(font)
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
	if err := extractLayoutInfo(otf); err != nil {
		return nil, err
	}
	// Collect and centralize font information:
	// The number of glyphs in the font is restricted only by the value stated in the 'head' table. The order in which glyphs are placed in a font is arbitrary.
	// Note that a font must have at least two glyphs, and that glyph index 0 musthave an outline. See Glyph Mappings for details.
	//
	if hh := otf.tables[T("hhea")]; hh != nil {
		hhead := hh.Self().AsHHea()
		if mx := otf.tables[T("hmtx")]; mx != nil {
			hmtx := mx.Self().AsHMtx()
			hmtx.NumberOfHMetrics = hhead.NumberOfHMetrics
		}
	}
	if he := otf.Table(T("head")); he != nil {
		head := he.Self().AsHead()
		if lo := otf.Table(T("loca")); lo != nil {
			loca := lo.Self().AsLoca()
			if head.IndexToLocFormat == 1 {
				loca.inx2loc = longLocaVersion
			}
			if ma := otf.Table(T("maxp")); ma != nil {
				maxp := ma.Self().AsMaxP()
				loca.locCnt = maxp.NumGlyphs
			}
		}
	}
	return otf, nil
}

// According to the OpenType spec, the following tables are
// required for the font to function correctly.
var RequiredTables = []string{
	"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post",
}

// These are the OpenType tables for advanced layout.
var LayoutTables = []string{
	"GSUB", "GPOS", "GDEF",
	//"GSUB", "GPOS", "GDEF", "BASE", "JSTF",
}

// Consistency check and shortcuts to essential tables, including layout tables.
func extractLayoutInfo(otf *Font) error {
	for _, tag := range RequiredTables {
		h := otf.tables[T(tag)]
		if h == nil {
			return errFontFormat("missing required table " + tag)
		}
	}
	otf.CMap = otf.tables[T("cmap")].Self().AsCMap()
	// We'll operate on OpenType fonts only, i.e. fonts containing GSUB and GPOS tables.
	for _, tag := range LayoutTables {
		h := otf.tables[T(tag)]
		if h == nil {
			return errFontFormat("missing advanced layout table " + tag)
		}
	}
	// store shortcuts to layout tables
	otf.Layout.GSub = otf.tables[T("GSUB")].Self().AsGSub()
	otf.Layout.GPos = otf.tables[T("GPOS")].Self().AsGPos()
	otf.Layout.GDef = otf.tables[T("GDEF")].Self().AsGDef()
	//otf.Layout.Base = otf.tables[T("BASE")].Self().AsBase()
	//otf.Layout.Jstf = otf.tables[T("JSTF")].Self().AsJstf()
	return nil
}

func parseTable(t Tag, b binarySegm, offset, size uint32) (Table, error) {
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
	tracer().Infof("font contains table (%s), will not be interpreted", t)
	return newTable(t, b, offset, size), nil
}

// --- Head table ------------------------------------------------------------

func parseHead(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
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
func parseBase(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
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
		tracer().Errorf("error parsing BASE table: %v", err)
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
		tracer().Debugf("axis table %d has %d entries", hOrV,
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

		tracer().Debugf("axis table %d has %d entries", hOrV,
			base.axisTables[hOrV].baselineTags.Count)
	}
	tracer().Infof("BASE axis %d has no/unreadable entires", hOrV)
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
//
//	Macintosh platformID are only required for backwards compatibility.”
//
// and:
// “The Unicode platform's platform-specific ID 6 was intended to mark a 'cmap' subtable
//
//	as one used by a last resort font. This is not required by any Apple platform.”
//
// All in all, we only support the following plaform/encoding/format combinations:
//
//	0 (Unicode)  3    4   Unicode BMB
//	0 (Unicode)  4    12  Unicode full
//	3 (Win)      1    4   Unicode BMP
//	3 (Win)      10   12  Unicode full
func parseCMap(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
	n, _ := b.u16(2) // number of sub-tables
	tracer().Debugf("font cmap has %d sub-tables in %d|%d bytes", n, len(b), size)
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
			tracer().Infof("cmap sub-table cannot be parsed")
			continue
		}
		subtable := link.Jump()
		format := subtable.U16(0)
		tracer().Debugf("cmap table contains subtable with format %d", format)
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
func parseKern(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
	if size <= 4 {
		return nil, nil
	}
	var N, suboffset, subheaderlen int
	if version := u32(b); version == 0x00010000 {
		tracer().Debugf("font has Apple TTF kern table format")
		n, _ := b.u32(4) // number of kerning tables is uint32
		N, suboffset, subheaderlen = int(n), 8, 16
	} else {
		tracer().Debugf("font has OTF (MS) kern table format")
		n, _ := b.u16(2) // number of kerning tables is uint16
		N, suboffset, subheaderlen = int(n), 4, 14
	}
	tracer().Debugf("kern table has %d sub-tables", N)
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
			tracer().Infof("kern sub-table format %d not supported, ignoring sub-table", format)
			continue // we only support format 0 kerning tables; skip this one
		}
		h.directory = [4]uint16{
			u16(b[suboffset+subheaderlen-8:]),
			u16(b[suboffset+subheaderlen-6:]),
			u16(b[suboffset+subheaderlen-4:]),
			u16(b[suboffset+subheaderlen-2:]),
		}
		kerncnt := uint32(h.directory[0])
		tracer().Debugf("kern sub-table has %d entries", kerncnt)
		// For some fonts, size calculation of kern sub-tables is off; see
		// https://github.com/fonttools/fonttools/issues/314#issuecomment-118116527
		// Testable with the Calibri font.
		sz := kerncnt * 6 // kern pair is of size 6
		if sz != h.length {
			tracer().Infof("kern sub-table size should be 0x%x, but given as 0x%x; fixing",
				sz, h.length)
		}
		if uint32(suboffset)+sz >= size {
			return nil, errFontFormat("kern sub-table size exceeds kern table bounds")
		}
		t.headers = append(t.headers, h)
		suboffset += int(subheaderlen + int(h.length))
	}
	tracer().Debugf("table kern has %d sub-table(s)", len(t.headers))
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
func parseLoca(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
	return newLocaTable(tag, b, offset, size), nil
}

// --- MaxP table ------------------------------------------------------------

// This table establishes the memory requirements for this font. Fonts with CFF data
// must use Version 0.5 of this table, specifying only the numGlyphs field. Fonts
// with TrueType outlines must use Version 1.0 of this table, where all data is required.
func parseMaxP(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
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
func parseHHea(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
	if size == 0 {
		return nil, nil
	}
	tracer().Debugf("HHea table has size %d", size)
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
func parseHMtx(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
	if size == 0 {
		return nil, nil
	}
	t := newHMtxTable(tag, b, offset, size)
	return t, nil
}

// --- Names -----------------------------------------------------------------

func parseNames(b binarySegm) (nameNames, error) {
	if len(b) < 6 {
		return nameNames{}, errFontFormat("name section corrupt")
	}
	N, _ := b.u16(2)
	names := nameNames{}
	strOffset, _ := b.u16(4)
	names.strbuf = b[strOffset:]
	tracer().Debugf("name table has %d strings, starting at %d", N, strOffset)
	if len(b) < 6+12*int(N) {
		return nameNames{}, errFontFormat("name section corrupt")
	}
	recs := b[6 : 6+12*int(N)]
	names.nameRecs = viewArray(recs, 12)
	return names, nil
}

// --- GDEF table ------------------------------------------------------------

// The Glyph Definition (GDEF) table provides various glyph properties used in
// OpenType Layout processing.
func parseGDef(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
	var err error
	gdef := newGDefTable(tag, b, offset, size)
	err = parseGDefHeader(gdef, b, err)
	err = parseGlyphClassDefinitions(gdef, b, err)
	err = parseAttachmentPointList(gdef, b, err)
	// we currently do not parse the Ligature Caret List Table
	err = parseMarkAttachmentClassDef(gdef, b, err)
	err = parseMarkGlyphSets(gdef, b, err)
	if err != nil {
		tracer().Errorf("error parsing GDEF table: %v", err)
		return gdef, err
	}
	mj, mn := gdef.Header().Version()
	tracer().Debugf("GDEF table has version %d.%d", mj, mn)
	return gdef, err
}

// The GDEF table begins with a header that starts with a version number. Three
// versions are defined. Version 1.0 contains an offset to a Glyph Class Definition
// table (GlyphClassDef), an offset to an Attachment List table (AttachList), an offset
// to a Ligature Caret List table (LigCaretList), and an offset to a Mark Attachment
// Class Definition table (MarkAttachClassDef). Version 1.2 also includes an offset to
// a Mark Glyph Sets Definition table (MarkGlyphSetsDef). Version 1.3 also includes an
// offset to an Item Variation Store table.
func parseGDefHeader(gdef *GDefTable, b binarySegm, err error) error {
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
func parseGlyphClassDefinitions(gdef *GDefTable, b binarySegm, err error) error {
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
func parseAttachmentPointList(gdef *GDefTable, b binarySegm, err error) error {
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
func parseMarkAttachmentClassDef(gdef *GDefTable, b binarySegm, err error) error {
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
func parseMarkGlyphSets(gdef *GDefTable, b binarySegm, err error) error {
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
func parseGPos(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
	var err error
	gpos := newGPosTable(tag, b, offset, size)
	err = parseLayoutHeader(&gpos.LayoutTable, b, err)
	err = parseLookupList(&gpos.LayoutTable, b, err)
	err = parseFeatureList(&gpos.LayoutTable, b, err)
	err = parseScriptList(&gpos.LayoutTable, b, err)
	if err != nil {
		tracer().Errorf("error parsing GPOS table: %v", err)
		return gpos, err
	}
	mj, mn := gpos.header.Version()
	tracer().Debugf("GPOS table has version %d.%d", mj, mn)
	tracer().Debugf("GPOS table has %d lookup list entries", gpos.LookupList.length)
	return gpos, err
}

// --- GSUB table ------------------------------------------------------------

// The Glyph Substitution (GSUB) table provides data for substition of glyphs for
// appropriate rendering of scripts, such as cursively-connecting forms in Arabic script,
// or for advanced typographic effects, such as ligatures.
func parseGSub(tag Tag, b binarySegm, offset, size uint32) (Table, error) {
	var err error
	gsub := newGSubTable(tag, b, offset, size)
	err = parseLayoutHeader(&gsub.LayoutTable, b, err)
	err = parseLookupList(&gsub.LayoutTable, b, err)
	err = parseFeatureList(&gsub.LayoutTable, b, err)
	err = parseScriptList(&gsub.LayoutTable, b, err)
	if err != nil {
		tracer().Errorf("error parsing GSUB table: %v", err)
		return gsub, err
	}
	mj, mn := gsub.header.Version()
	tracer().Debugf("GSUB table has version %d.%d", mj, mn)
	tracer().Debugf("GSUB table has %d lookup list entries", gsub.LookupList.length)
	return gsub, err
}

// === Common Code for GPOS and GSUB =========================================

// parseLayoutHeader parses a layout table header, i.e. reads version information
// and header information (containing offsets).
// Supports header versions 1.0 and 1.1
func parseLayoutHeader(lytt *LayoutTable, b binarySegm, err error) error {
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

// --- Script list -----------------------------------------------------------

// A ScriptList table consists of a count of the scripts represented by the glyphs in the
// font (ScriptCount) and an array of records (ScriptRecord), one for each script for which
// the font defines script-specific features (a script without script-specific features
// does not need a ScriptRecord). Each ScriptRecord consists of a ScriptTag that identifies
// a script, and an offset to a Script table. The ScriptRecord array is stored in
// alphabetic order of the script tags.
func parseScriptList(lytt *LayoutTable, b binarySegm, err error) error {
	if err != nil {
		return err
	}
	//lytt.ScriptList = tagRecordMap16{}
	link := link16{base: b, offset: uint16(lytt.header.offsetFor(layoutScriptSection))}
	scripts := link.Jump() // now we stand at the ScriptList table
	//scriptRecords := parseTagRecordMap16(scripts.Bytes(), 0, scripts.Bytes(), "ScriptList", "Script")
	//lytt.ScriptList = scriptRecords
	lytt.ScriptList = NavigatorFactory("ScriptList", scripts, scripts)
	return nil
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
	//lytt.FeatureList = array{}
	lytt.FeatureList = tagRecordMap16{}
	link := link16{base: b, offset: uint16(lytt.header.offsetFor(layoutFeatureSection))}
	features := link.Jump() // now we stand at the FeatureList table
	featureRecords := parseTagRecordMap16(features.Bytes(), 0, features.Bytes(), "FeatureList", "Feature")
	//featureRecords, err := parseArray(features.Bytes(), 0, 6, "FeatureList", "Feature")
	if err != nil {
		return err
	}
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
func parseLangSys(b binarySegm, offset int, target string) (langSys, error) {
	lsys := langSys{}
	if len(b) < offset+4 {
		return lsys, errBufferBounds
	}
	tracer().Debugf("parsing LangSys (%s)", target)
	b = b[offset:]
	lsys.mandatory, _ = b.u16(0)
	features, err := parseArray16(b, 2, "LangSys", target)
	if err != nil {
		return lsys, err
	}
	lsys.featureIndices = features
	tracer().Debugf("LangSys points to %d features", features.length)
	return lsys, nil
}

// --- Layout table lookup list ----------------------------------------------

// parseLookupList parses the LookupList.
// See https://www.microsoft.com/typography/otspec/chapter2.htm#lulTbl
func parseLookupList(lytt *LayoutTable, b binarySegm, err error) error {
	if err != nil {
		return err
	}
	lloffset := lytt.header.offsetFor(layoutLookupSection)
	if lloffset >= len(b) {
		return io.ErrUnexpectedEOF
	}
	b = b[lloffset:]
	//
	ll := LookupList{base: b}
	ll.array, ll.err = parseArray16(b, 0, "Lookup", "Lookup-Subtales")
	lytt.LookupList = ll
	return nil
}

func parseLookupSubtable(b binarySegm, lookupType LayoutTableLookupType) LookupSubtable {
	tracer().Debugf("parse lookup subtable b = %v", asU16Slice(b[:20]))
	if len(b) < 4 {
		return LookupSubtable{}
	}
	if IsGPosLookupType(lookupType) {
		return parseGPosLookupSubtable(b, GPosLookupType(lookupType))
	}
	return parseGSubLookupSubtable(b, GSubLookupType(lookupType))
}

// parseGSubLookupSubtable parses a segment of binary data from a font file (NavLocation)
// and expects to read a lookup subtable.
func parseGSubLookupSubtable(b binarySegm, lookupType LayoutTableLookupType) LookupSubtable {
	//trace().Debugf("parse lookup subtable b = %v", asU16Slice(b[:20]))
	format := b.U16(0)
	tracer().Debugf("parsing GSUB sub-table type %s, format %d", lookupType.GSubString(), format)
	sub := LookupSubtable{LookupType: lookupType, Format: format}
	// Most of the subtable formats use a coverage table in some form to decide on which glyphs to
	// operate on. parseGSubLookupSubtable will parse this coverage table and put it into
	// `sub.Coverage`, then branch down to the different lookup types.
	if !(lookupType == 7 && format == 3) { // GSUB type Extension has no coverage table
		covlink, _ := parseLink16(b, 2, b, "Coverage")
		sub.Coverage = parseCoverage(covlink.Jump().Bytes())
	}
	switch lookupType {
	case 1:
		return parseGSubLookupSubtableType1(b, sub)
	case 2, 3, 4:
		return parseGSubLookupSubtableType2or3or4(b, sub)
	case 5:
		return parseGSubLookupSubtableType5(b, sub)
	case 6:
		return parseGSubLookupSubtableType6(b, sub)
	case 7:
		return parseGSubLookupSubtableType7(b, sub)
	}
	tracer().Errorf("unknown GSUB lookup type: %d", lookupType)
	return LookupSubtable{}
}

// LookupType 1: Single Substitution Subtable
// Single substitution (SingleSubst) subtables tell a client to replace a single glyph with
// another glyph.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-1-single-substitution-subtable
func parseGSubLookupSubtableType1(b binarySegm, sub LookupSubtable) LookupSubtable {
	if sub.Format == 1 {
		sub.Support = int16(b.U16(4))
	} else {
		sub.Index = parseVarArray16(b, 4, 2, 1, "LookupSubtableGSub1")
	}
	return sub
}

// LookupType 2: Multiple Substitution Subtable
// A Multiple Substitution (MultipleSubst) subtable replaces a single glyph with more than
// one glyph, as when multiple glyphs replace a single ligature.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-2-multiple-substitution-subtable
//
// LookupType 3: Alternate Substitution Subtable
// An Alternate Substitution (AlternateSubst) subtable identifies any number of aesthetic
// alternatives from which a user can choose a glyph variant to replace the input glyph.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-3-alternate-substitution-subtable
//
// LookupType 4: Ligature Substitution Subtable
// A Ligature Substitution (LigatureSubst) subtable identifies ligature substitutions where
// a single glyph replaces multiple glyphs. One LigatureSubst subtable can specify any
// number of ligature substitutions.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-4-ligature-substitution-subtable
func parseGSubLookupSubtableType2or3or4(b binarySegm, sub LookupSubtable) LookupSubtable {
	sub.Index = parseVarArray16(b, 4, 2, 2, "LookupSubtableGSub2/3/4")
	return sub
}

// LookupType 5: Contextual Substitution Subtable
// A Contextual Substitution subtable describes glyph substitutions in context that replace one or more
// glyphs within a certain pattern of glyphs.
// Input sequence patterns are matched against the text glyph sequence, and then actions to be applied
// to glyphs within the input sequence. The actions are specified as “nested” lookups, and each is applied
// to a particular sequence position within the input sequence.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-5-contextual-substitution-subtable
//
// For contextual substitution subtables we usually will have to parse a rule set. We will put it
// into the Index field. Additional context data structures may include ClassDefs or other things and
// will be put into the Support field by calling parseSequenceContext.
func parseGSubLookupSubtableType5(b binarySegm, sub LookupSubtable) LookupSubtable {
	switch sub.Format {
	case 1:
		sub.Index = parseVarArray16(b, 4, 2, 2, "LookupSubtableGSub5-1")
	case 2:
		sub.Index = parseVarArray16(b, 6, 2, 2, "LookupSubtableGSub5-2")
	case 3:
		sub.Index = parseVarArray16(b, 4, 4, 2, "LookupSubtableGSub5-3")
	}
	var err error
	sub, err = parseSequenceContext(b, sub)
	if err != nil {
		tracer().Errorf(err.Error()) // nothing we can/will do about it
	}
	return sub
}

// LookupType 6: Chained Contexts Substitution Subtable
// A Chained Contexts Substitution subtable describes glyph substitutions in context with an ability to
// look back and/or look ahead in the sequence of glyphs. The design of the Chained Contexts Substitution
// subtable is parallel to that of the Contextual Substitution subtable, including the availability of
// three formats. Each format can describe one or more chained backtrack, input, and lookahead sequence
// combinations, and one or more substitutions for glyphs in each input sequence.
// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#chained-sequence-context-format-1-simple-glyph-contexts
func parseGSubLookupSubtableType6(b binarySegm, sub LookupSubtable) LookupSubtable {
	var err error
	sub, err = parseChainedSequenceContext(b, sub)
	if err != nil {
		tracer().Errorf(err.Error()) // nothing we can/will do about it
	}
	switch sub.Format {
	case 1:
		sub.Index = parseVarArray16(b, 4, 2, 2, "LookupSubtableGSub6-1")
	case 2:
		sub.Index = parseVarArray16(b, 10, 2, 2, "LookupSubtableGSub6-2")
	case 3:
		offset := 2 // skip over format field
		// TODO treat error conditions
		seqctx := sub.Support.(*SequenceContext)
		offset += 2 + len(seqctx.BacktrackCoverage)*2
		offset += 2 + len(seqctx.InputCoverage)*2
		offset += 2 + len(seqctx.LookaheadCoverage)*2
		sub.Index = parseVarArray16(b, offset, 2, 2, "LookupSubtableGSub6-3")
	}
	return sub
}

// LookupType 7: Extension Substitution
// This lookup provides a mechanism whereby any other lookup type’s subtables are stored at
// a 32-bit offset location in the GSUB table.
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-7-extension-substitution
func parseGSubLookupSubtableType7(b binarySegm, sub LookupSubtable) LookupSubtable {
	if b.Size() < 8 {
		tracer().Errorf("OpenType GSUB lookup subtable type %d corrupt", sub.LookupType)
		return LookupSubtable{}
	}
	if sub.LookupType = LayoutTableLookupType(b.U16(2)); sub.LookupType == GSubLookupTypeExtensionSubs {
		tracer().Errorf("OpenType GSUB lookup subtable type 7 recursion detected")
		return LookupSubtable{}
	}
	tracer().Debugf("OpenType GSUB extension subtable is of type %s", sub.LookupType.GSubString())
	link, _ := parseLink32(b, 4, b, "ext.LookupSubtable")
	loc := link.Jump()
	return parseGSubLookupSubtable(loc.Bytes(), sub.LookupType)
}

func parseGPosLookupSubtable(b binarySegm, lookupType LayoutTableLookupType) LookupSubtable {
	format := b.U16(0)
	tracer().Debugf("parsing GPOS sub-table type %s, format %d", lookupType.GPosString(), format)
	panic("TODO GPOS Lookup Subtable")
	//return LookupSubtable{}
}

// --- parse class def table -------------------------------------------------

// The ClassDef table can have either of two formats: one that assigns a range of
// consecutive glyph indices to different classes, or one that puts groups of consecutive
// glyph indices into the same class.
func parseClassDefinitions(b binarySegm) (ClassDefinitions, error) {
	tracer().Debugf("HELLO, parsing a ClassDef")
	cdef := ClassDefinitions{}
	r := bytes.NewReader(b)
	if err := binary.Read(r, binary.BigEndian, &cdef.format); err != nil {
		return cdef, err
	}
	var n, g uint16
	if cdef.format == 1 {
		tracer().Debugf("parsing a ClassDef of format 1")
		n, _ = b.u16(4) // number of glyph IDs in table
		g, _ = b.u16(2) // start glyph ID
	} else if cdef.format == 2 {
		tracer().Debugf("parsing a ClassDef of format 2")
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
func parseCoverage(b binarySegm) Coverage {
	tracer().Debugf("parsing Coverage")
	h := coverageHeader{}
	h.CoverageFormat = b.U16(0)
	h.Count = b.U16(2)
	// r := bytes.NewReader(b)
	// if err := binary.Read(r, binary.BigEndian, &h); err != nil {
	// 	return Coverage{}
	// }
	tracer().Debugf("coverage header format %d has count = %d ", h.CoverageFormat, h.Count)
	//trace().Debugf("cont = %v", asU16Slice(b[:20]))
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
func parseSequenceContext(b binarySegm, sub LookupSubtable) (LookupSubtable, error) {
	if len(b) <= 2 {
		return sub, errFontFormat("corrupt sequence context")
	}
	//format := b.U16(0)
	switch sub.Format {
	case 1:
		return parseSequenceContextFormat1(b, sub)
	case 2:
		return parseSequenceContextFormat2(b, sub)
	case 3:
		return parseSequenceContextFormat3(b, sub)
	}
	return sub, errFontFormat(fmt.Sprintf("unknown sequence context format %d", sub.Format))
}

// SequenceContextFormat1: simple glyph contexts
// Type 	Name 	Description
// uint16 	format 	Format identifier: format = 1
// Offset16 	coverageOffset 	Offset to Coverage table, from beginning of SequenceContextFormat1 table
// uint16 	seqRuleSetCount 	Number of SequenceRuleSet tables
// Offset16 	seqRuleSetOffsets[seqRuleSetCount] 	Array of offsets to SequenceRuleSet tables, from beginning of SequenceContextFormat1 table (offsets may be NULL)
func parseSequenceContextFormat1(b binarySegm, sub LookupSubtable) (LookupSubtable, error) {
	if len(b) <= 6 {
		return sub, errFontFormat("corrupt sequence context")
	}
	// nothing to to for format 1
	//
	// seqctx := SequenceContext{}
	// link, err := parseLink16(b, 2, b, "SequenceContext Coverage")
	// if err != nil {
	// 	return sequenceContext{}, errFontFormat("corrupt sequence context")
	// }
	// cov := link.Jump()
	// seqctx.coverage[0] = parseCoverage(cov.Bytes())
	// seqctx.rules = parseVarArrary16(b, 4, 2, "SequenceContext")
	// return seqctx, nil
	return sub, nil
}

// SequenceContextFormat2 table:
// Type      Name                   Description
// uint16    format                 Format identifier: format = 2
// Offset16  coverageOffset         Offset to Coverage table, from beginning of SequenceContextFormat2 table
// Offset16  classDefOffset         Offset to ClassDef table, from beginning of SequenceContextFormat2 table
// uint16    classSeqRuleSetCount   Number of ClassSequenceRuleSet tables
// Offset16  classSeqRuleSetOffsets[classSeqRuleSetCount]    Array of offsets to ClassSequenceRuleSet tables, from beginning of SequenceContextFormat2 table (may be NULL)
func parseSequenceContextFormat2(b binarySegm, sub LookupSubtable) (LookupSubtable, error) {
	if len(b) <= 8 {
		return sub, errFontFormat("corrupt sequence context")
	}
	seqctx := &SequenceContext{}
	sub.Support = seqctx
	seqctx.ClassDefs = make([]ClassDefinitions, 1)
	var err error
	seqctx.ClassDefs[0], err = parseContextClassDef(b, 4)
	sub.Support = seqctx
	return sub, err
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
func parseSequenceContextFormat3(b binarySegm, sub LookupSubtable) (LookupSubtable, error) {
	if len(b) <= 8 {
		return sub, errFontFormat("corrupt sequence context")
	}
	glyphCount := int(b.U16(2))
	seqctx := SequenceContext{}
	sub.Support = seqctx
	seqctx.InputCoverage = make([]Coverage, glyphCount)
	for i := 0; i < glyphCount; i++ {
		link, err := parseLink16(b, 6+i*2, b, "SequenceContext Coverage")
		if err != nil {
			return sub, errFontFormat("corrupt sequence context")
		}
		cov := link.Jump()
		seqctx.InputCoverage[i] = parseCoverage(cov.Bytes())
	}
	return sub, nil
}

func parseChainedSequenceContext(b binarySegm, sub LookupSubtable) (LookupSubtable, error) {
	if len(b) <= 2 {
		return sub, errFontFormat("corrupt chained sequence context")
	}
	switch sub.Format {
	case 1:
		//parseSequenceContextFormat1(sub.Format, b, sub)
		// nothing to to for format 1
		panic("TODO chained 1")
		//return sub, nil
	case 2:
		return parseChainedSequenceContextFormat2(b, sub)
	case 3:
		return parseChainedSequenceContextFormat3(b, sub)
	}
	return sub, errFontFormat(fmt.Sprintf("unknown chained sequence context format %d", sub.Format))
}

func parseChainedSequenceContextFormat2(b binarySegm, sub LookupSubtable) (LookupSubtable, error) {
	backtrack, err1 := parseContextClassDef(b, 4)
	input, err2 := parseContextClassDef(b, 6)
	lookahead, err3 := parseContextClassDef(b, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return LookupSubtable{}, errFontFormat("corrupt chained sequence context (format 2)")
	}
	sub.Support = &SequenceContext{
		ClassDefs: []ClassDefinitions{backtrack, input, lookahead},
	}
	return sub, nil
}

func parseChainedSequenceContextFormat3(b binarySegm, sub LookupSubtable) (LookupSubtable, error) {
	tracer().Debugf("chained sequence context format 3 ........................")
	tracer().Debugf("b = %v", b[:26].Glyphs())
	offset := 2
	backtrack, err1 := parseChainedSeqContextCoverages(b, offset, nil)
	offset += 2 + len(backtrack)*2
	input, err2 := parseChainedSeqContextCoverages(b, offset, err1)
	offset += 2 + len(input)*2
	lookahead, err3 := parseChainedSeqContextCoverages(b, offset, err2)
	if err1 != nil || err2 != nil || err3 != nil {
		return LookupSubtable{}, errFontFormat("corrupt chained sequence context (format 3)")
	}
	sub.Support = &SequenceContext{
		BacktrackCoverage: backtrack,
		InputCoverage:     input,
		LookaheadCoverage: lookahead,
	}
	return sub, nil
}

func parseContextClassDef(b binarySegm, at int) (ClassDefinitions, error) {
	link, err := parseLink16(b, at, b, "ClassDef")
	if err != nil {
		return ClassDefinitions{}, err
	}
	cdef, err := parseClassDefinitions(link.Jump().Bytes())
	if err != nil {
		return ClassDefinitions{}, err
	}
	return cdef, nil
}

func parseChainedSeqContextCoverages(b binarySegm, at int, err error) ([]Coverage, error) {
	if err != nil {
		return []Coverage{}, err
	}
	count := int(b.U16(at))
	coverages := make([]Coverage, count)
	tracer().Debugf("chained seq context with %d coverages", count)
	for i := 0; i < count; i++ {
		link, err := parseLink16(b, at+2+i*2, b, "ChainedSequenceContext Coverage")
		if err != nil {
			tracer().Errorf("error parsing coverages' offset")
			return []Coverage{}, err
		}
		coverages[i] = parseCoverage(link.Jump().Bytes())
	}
	return coverages, nil
}

// TODO Argument should be NavLocation, return value should be []SeqLookupRecord
//
// SequenceRule table:
// Type     Name                          Description
// uint16   glyphCount                    Number of glyphs to be matched
// uint16   seqLookupCount                Number of SequenceLookupRecords
// uint16   inputSequence[glyphCount-1]   Sequence of classes to be matched to the input glyph sequence, beginning with the second glyph position
// SequenceLookupRecord seqLookupRecords[seqLookupCount]   Array of SequenceLookupRecords
func (lksub LookupSubtable) SequenceRule(b binarySegm) sequenceRule {
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
