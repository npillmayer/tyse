package ot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ConradIrwin/font/sfnt"
	"github.com/npillmayer/tyse/core"
)

func errFontFormat(x string) error {
	return core.Error(core.EINVALID, "OpenType font format: %s", x)
}

// Parse parses an OpenType font from a byte slice.
func Parse(font []byte) (*OTFont, error) {
	// https://www.microsoft.com/typography/otspec/otff.htm: Offset Table is 12 bytes.
	r := bytes.NewReader(font)
	h := fontHeader{}
	if err := binary.Read(r, binary.BigEndian, &h); err != nil {
		return nil, err
	}
	trace().Debugf("header = %v, tag = %x|%s", h, h.FontType, Tag(h.FontType).String())
	if !(h.FontType == 0x4f54544f || // OTTO
		h.FontType == 0x00010000 || // TrueType
		h.FontType == 0x74727565) { // true
		return nil, errFontFormat(fmt.Sprintf("font type not supported: %x", h.FontType))
	}
	otf := &OTFont{header: &h, tables: make(map[Tag]Table)}
	src := fontBinSegm(font)
	// "The Offset Table is followed immediately by the Table Record entries …
	// sorted in ascending order by tag", 16 bytes each.
	buf, err := src.view(12, 16*int(h.TableCount))
	if err != nil {
		return nil, errFontFormat("table record entries")
	}
	for b, prevTag := buf, Tag(0); len(b) > 0; b = b[16:] {
		tag := tag(b)
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
	return otf, nil
}

func parseTable(t Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	switch t {
	case tag([]byte("name")), tag([]byte("DSIG")), tag([]byte("Feat")):
		return nil, nil // currently not needed / supported
	case tag([]byte("head")):
		return parseHead(t, b, offset, size)
	case tag([]byte("cmap")):
		return newTable(t, b, offset, size), nil
	case tag([]byte("GSUB")):
		return parseGSub(t, b, offset, size)
	case tag([]byte("GPOS")):
		return newTable(t, b, offset, size), nil
	case tag([]byte("kern")):
		return newTable(t, b, offset, size), nil
	case tag([]byte("CBLC")), tag([]byte("OS/2")), tag([]byte("math")), tag([]byte("hhea")),
		tag([]byte("GDEF")):
		return newTable(t, b, offset, size), nil
	}
	return nil, errFontFormat(fmt.Sprintf("unsupported table tag: %s", t))
}

// --- Head table ------------------------------------------------------------

type HeadTable struct {
	TableBase
	flags      uint16
	unitsPerEm uint16
}

func (t *HeadTable) Base() *TableBase {
	return &t.TableBase
}

func parseHead(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	if size < 54 {
		return nil, errFontFormat("size of head table")
	}
	f, _ := b.u16(16)
	u, _ := b.u16(18)
	t := &HeadTable{flags: f, unitsPerEm: u}
	t.self = t
	return t, nil
}

// --- CMap table ------------------------------------------------------------

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

func parseCMap(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	n, _ := b.u16(2)
	t := &CMapTable{
		numTables: int(n),
	}
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

// TODO
func platformEncodingWidth(pid, psid uint16) int {
	return 7
}

// --- GSUB table ------------------------------------------------------------

func parseGSub(tag Tag, b fontBinSegm, offset, size uint32) (Table, error) {
	var err error
	gsub := &GSubTable{}
	err = parseLayoutHeader(gsub, b, err)
	err = parseLookupList(gsub, b, err)
	//err = parseFeatureList(otf, gsub, err)
	//err = parseScriptList(otf, gsub, err)
	if err != nil {
		trace().Errorf("error parsing GSUB table: %v", err)
		return gsub, err
	}
	mj, mn := gsub.header.Version()
	trace().Debugf("GSUB table has version %d.%d", mj, mn)
	trace().Debugf("GSUB table has %d lookup list entries", len(gsub.lookups))
	return gsub, err
}

// --- Layout table header ---------------------------------------------------

// LayoutHeader represents header information for layout tables, i.e.
// GPOS and GSUB.
type LayoutHeader struct {
	version versionHeader
	offsets layoutHeader11
}

// Version returns major and minor version number for this layout table.
func (lh *LayoutHeader) Version() (int, int) {
	return int(lh.version.Major), int(lh.version.Minor)
}

// Offset returns an offset for a layout table section
func (lh *LayoutHeader) Offset(which LayoutTableSectionName) int {
	switch which {
	case LayoutScriptSection:
		return int(lh.offsets.ScriptListOffset)
	case LayoutFeatureSection:
		return int(lh.offsets.FeatureListOffset)
	case LayoutLookupSection:
		return int(lh.offsets.LookupListOffset)
	case LayoutFeatureVariationsSection:
		return int(lh.offsets.FeatureVariationsOffset)
	}
	return 0 // cannot happen
}

// versionHeader is the beginning of on-disk format of the GPOS/GSUB version header.
// See https://www.microsoft.com/typography/otspec/GPOS.htm
// See https://www.microsoft.com/typography/otspec/GSUB.htm
// Fields are public for reflection-access.
type versionHeader struct {
	Major uint16 // Major version of the GPOS/GSUB table.
	Minor uint16 // Minor version of the GPOS/GSUB table.
}

// layoutHeader10 is the on-disk format of GPOS/GSUB version header when major=1 and minor=0.
// Fields are public for reflection-access.
type layoutHeader10 struct {
	ScriptListOffset  uint16 // offset to ScriptList table, from beginning of GPOS/GSUB table.
	FeatureListOffset uint16 // offset to FeatureList table, from beginning of GPOS/GSUB table.
	LookupListOffset  uint16 // offset to LookupList table, from beginning of GPOS/GSUB table.
}

// layoutHeader11 is the on-disk format of GPOS/GSUB version header when major=1 and minor=1.
// Fields are public for reflection-access.
type layoutHeader11 struct {
	layoutHeader10
	FeatureVariationsOffset uint32 // offset to FeatureVariations table, from beginning of GPOS/GSUB table (may be NULL).
}

// parseLayoutHeader parses a layout table header, i.e. reads version information
// and header information (containing offsets).
// Supports header versions 1.0 and 1.1
func parseLayoutHeader(gsub *GSubTable, b []byte, err error) error {
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
	gsub.header = h
	return nil
}

// --- GSUB lookup list ------------------------------------------------------

type lookupRecord struct {
	lookupRecordInfo
	subrecordOffsets []uint16 // Array of offsets to lookup subrecords, from beginning of Lookup table
	// markFilteringSet uint16 // Index (base 0) into GDEF mark glyph sets structure. This field is only present if bit useMarkFilteringSet of lookup flags is set.
}

type lookupRecordInfo struct {
	Type           uint16
	Flag           uint16 // Lookup qualifiers
	SubRecordCount uint16 // Number of subrecords for this lookup
}

type lookupSubstFormat1 struct {
	Format         uint16 // 1
	CoverageOffset uint16 // Offset to Coverage table, from beginning of substitution subtable
	Glyphs         int16  // Changes meaning, depending on format 1 or 2
}

type lookupSubstFormat2 struct {
	lookupSubstFormat1
	SubstituteGlyphIDs []uint16
}

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
	trace().Debugf("lookup table %s has %d subtables", GSubLookupTypeString(lookup.Type), lookup.SubRecordCount)
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
		trace().Debugf("offset of sub-table[%d] = %d", i, subs[i])
		r = bytes.NewReader(b[offset+off:])
		subst := lookupSubstFormat1{}
		if err := binary.Read(r, binary.BigEndian, &subst); err != nil {
			return nil, fmt.Errorf("reading lookupRecord: %s", err)
		}
		trace().Debugf("   format spec = %d", subst.Format)
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
func parseLookupList(gsub *GSubTable, b []byte, err error) error {
	if err != nil {
		return err
	}
	lloffset := gsub.header.Offset(LayoutLookupSection)
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
		// if err = read16arr(b, 2, &lookupOffsets, int(count)); err != nil {
		// 	return err
		// }
		gsub.lookups = nil
		for i := 0; i < int(count); i++ {
			//for i := 0; i < 1; i++ {
			trace().Debugf("lookup offset #%d = %d", i, lookupOffsets[i])
			// var record tagOffsetRecord
			// if err := binary.Read(r, binary.BigEndian, &record); err != nil {
			// 	return fmt.Errorf("reading lookupRecord[%d]: %s", i, err)
			// }
			lookup, err := parseLookup(b, lookupOffsets[i])
			if err != nil {
				return err
			}
			gsub.lookups = append(gsub.lookups, lookup)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------

func read16arr(r *bytes.Reader, arr *[]uint16, size int) error {
	*arr = make([]uint16, size, size)
	//r := bytes.NewReader(b[offset:])
	return binary.Read(r, binary.BigEndian, arr)
}

// ---------------------------------------------------------------------------

// The code below is partially replicated from github.com/ConradIrwin/font/sfnt because
// we need finer control of GPOS and GSUB tables and the fields are not exported or
// otherwise accessible.

// tagOffsetRecord is an on-disk format of a Tag and Offset record.
type tagOffsetRecord struct {
	Tag    sfnt.Tag // 4-byte script tag identifier
	Offset uint16   // Offset to object from beginning of list
}
