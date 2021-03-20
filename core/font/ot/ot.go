package ot

import (
	"github.com/ConradIrwin/font/sfnt"
	"github.com/npillmayer/tyse/core/font"
)

// Font represents the internal structure of an OpenType font.
// It is used to navigate properties of a font for typesetting tasks.
type Font struct {
	f          *font.ScalableFont
	ot         *sfnt.Font // TODO remove this
	Header     *FontHeader
	tables     map[Tag]Table
	glyphIndex glyphIndexFunc
}

// FontHeader is a directory of the top-level tables in a font. If the font file
// contains only one font, the table directory will begin at byte 0 of the file.
// If the font file is an OpenType Font Collection file (see below), the beginning
// point of the table directory for each font is indicated in the TTCHeader.
type FontHeader struct {
	// OpenType fonts that contain TrueType outlines should use the value of 0x00010000
	// for the FontType. OpenType fonts containing CFF data (version 1 or 2) should
	// use 0x4F54544F ('OTTO', when re-interpreted as a Tag).
	// The Apple specification for TrueType fonts allows for 'true' and 'typ1',
	// but these version tags should not be used for OpenType fonts.
	FontType   uint32
	TableCount uint16
}

// Table returns the font table for a given tag. If a table for a tag cannot
// be found in the font, nil is returned.
//
// Please note that the current implementation will not interpret every kind of
// font table, either because there is no need to do so (with regard to
// text shaping or rasterization), or because implementation is not yet finished.
// However, `Table` will return at least a generic table type for each table contained in
// the font, i.e. no table information will be dropped.
//
// For example to receive the `OS/2` and the `loca` table, clients may call
//
//    os2  := otf.Table(ot.T("OS/2"))
//    loca := otf.Table(ot.T("loca")).Base().AsLoca()
//
// Table tag names are case-sensitive, following the names in the OpenType specification,
// i.e., one of:
//
// avar BASE CBDT CBLC CFF CFF2 cmap COLR CPAL cvar cvt DSIG EBDT EBLC EBSC fpgm fvar
// gasp GDEF glyf GPOS GSUB gvar hdmx head hhea hmtx HVAR JSTF kern loca LTSH MATH
// maxp MERG meta MVAR name OS/2 PCLT post prep sbix STAT SVG VDMX vhea vmtx VORG VVAR
//
func (otf *Font) Table(tag Tag) Table {
	if t, ok := otf.tables[tag]; ok {
		return t
	}
	return nil
}

// TableTags returns a list of tags, one for each table contained in the font.
func (otf *Font) TableTags() []Tag {
	var tags = make([]Tag, 0, len(otf.tables))
	for tag := range otf.tables {
		tags = append(tags, tag)
	}
	return tags
}

// GlyphIndex is a glyph index in a font.
type GlyphIndex uint16

// GlyphIndex returns the glyph index for the given rune.
//
// It returns (0, nil) if there is no glyph for r.
// https://www.microsoft.com/typography/OTSPEC/cmap.htm says that "Character
// codes that do not correspond to any glyph in the font should be mapped to
// glyph index 0. The glyph at this location must be a special glyph
// representing a missing character, commonly known as .notdef."
func (otf *Font) GlyphIndex(codePoint rune) (GlyphIndex, error) {
	return otf.glyphIndex(otf, codePoint)
}

// --- Layout tables (GSUB & GPOS) -------------------------------------------

// LayoutTableSectionName lists the sections of OT layout tables, i.e. GPOS and GSUB.
type LayoutTableSectionName int

const (
	LayoutScriptSection LayoutTableSectionName = iota
	LayoutFeatureSection
	// Lookup records:
	// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#lookup-table
	LayoutLookupSection
	LayoutFeatureVariationsSection
)

// LayoutTableLookupFlag is a flag type for layout tables (GPOS and GSUB).
type LayoutTableLookupFlag uint16

// Lookup flags of layout tables (GPOS and GSUB)
const ( // LookupFlag bit enumeration
	// Note that the RIGHT_TO_LEFT flag is used only for GPOS type 3 lookups and is ignored
	// otherwise. It is not used by client software in determining text direction.
	LOOKUP_FLAG_RIGHT_TO_LEFT             LayoutTableLookupFlag = 0x0001
	LOOKUP_FLAG_IGNORE_BASE_GLYPHS        LayoutTableLookupFlag = 0x0002 // If set, skips over base glyphs
	LOOKUP_FLAG_IGNORE_LIGATURES          LayoutTableLookupFlag = 0x0004 // If set, skips over ligatures
	LOOKUP_FLAG_IGNORE_MARKS              LayoutTableLookupFlag = 0x0008 // If set, skips over all combining marks
	LOOKUP_FLAG_USE_MARK_FILTERING_SET    LayoutTableLookupFlag = 0x0010 // If set, indicates that the lookup table structure is followed by a MarkFilteringSet field.
	LOOKUP_FLAG_reserved                  LayoutTableLookupFlag = 0x00E0 // For future use (Set to zero)
	LOOKUP_FLAG_MARK_ATTACHMENT_TYPE_MASK LayoutTableLookupFlag = 0xFF00 // If not zero, skips over all marks of attachment type different from specified.
)

// LayoutTableLookupType is a type identifier for layout lookup records (GPOS and GSUB).
// Enum values are different for GPOS and GSUB.
type LayoutTableLookupType uint16

// Layout table script record
type scriptRecord struct {
	Tag    Tag
	Offset uint16
}

// Layout table feature record
type featureRecord struct {
	Tag    Tag
	Offset uint16
}

// Layout table lookup list
type lookupRecord struct {
	lookupRecordInfo
	subrecordOffsets []uint16 // Array of offsets to lookup subrecords, from beginning of Lookup table
	// markFilteringSet uint16 // Index (base 0) into GDEF mark glyph sets structure. This field is only present if bit useMarkFilteringSet of lookup flags is set.
}

type lookupRecordInfo struct {
	Type           LayoutTableLookupType
	Flag           LayoutTableLookupFlag
	SubRecordCount uint16 // Number of subrecords for this lookup
}

// --- Tag -------------------------------------------------------------------

// Tag is defined by the spec as:
// Array of four uint8s (length = 32 bits) used to identify a table, design-variation axis,
// script, language system, feature, or baseline
type Tag uint32

// MakeTag creates a Tag from 4 bytes, e.g.,
// If b is shorter or longer, it will be silently extended or cut as appropriate
//
//    MakeTag([]byte("cmap"))
//
func MakeTag(b []byte) Tag {
	if b == nil {
		b = []byte{0, 0, 0, 0}
	} else if len(b) > 4 {
		b = b[:4]
	} else if len(b) < 4 {
		b = append([]byte{0, 0, 0, 0}[:4-len(b)], b...)
	}
	return Tag(u32(b))
}

// T returns a Tag from a (4-letter) string.
// If t is shorter or longer, it will be silently extended or cut as appropriate
func T(t string) Tag {
	t = "    "[:4-len(t)] + t
	return Tag(u32([]byte(t)))
}

func (t Tag) String() string {
	bytes := []byte{
		byte(t >> 24 & 0xff),
		byte(t >> 16 & 0xff),
		byte(t >> 8 & 0xff),
		byte(t & 0xff),
	}
	return string(bytes)
}

// --- Table -----------------------------------------------------------------

// Table represents one of the various OpenType font tables
//
// Required Tables, according to the OpenType specification:
// 'cmap' (Character to glyph mapping), 'head' (Font header), 'hhea' (Horizontal header),
// 'hmtx' (Horizontal metrics), 'maxp' (Maximum profile), 'name' (Naming table),
// 'OS/2' (OS/2 and Windows specific metrics), 'post' (PostScript information).
//
// Advanced Typographic Tables: 'BASE' (Baseline data), 'GDEF' (Glyph definition data),
// 'GPOS' (Glyph positioning data), 'GSUB' (Glyph substitution data),
// 'JSTF' (Justification data), 'MATH' (Math layout data).
//
// For TrueType outline fonts: 'cvt ' (Control Value Table, optional),
// 'fpgm' (Font program, optional), 'glyf' (Glyph data), 'loca' (Index to location),
// 'prep' (CVT Program, optional), 'gasp' (Grid-fitting/Scan-conversion, optional).
//
// For OpenType fonts based on CFF outlines: 'CFF ' (Compact Font Format 1.0),
// 'CFF2' (Compact Font Format 2.0), 'VORG' (Vertical Origin, optional).
//
// Currently not used/supported:
// SVG font table, bitmap glyph tables, color font tables, font variations.
//
type Table interface {
	Offset() uint32   // offset within the font's binary data
	Len() uint32      // byte size of table
	Binary() []byte   // the bytes of this table; should be treatet as read-only by clients
	String() string   // 4-letter table name, e.g., "cmap"
	Base() *TableBase // every table we use will be derived from TableBase
}

func newTable(tag Tag, b fontBinSegm, offset, size uint32) *genericTable {
	t := &genericTable{TableBase{
		data:   b,
		name:   tag,
		offset: offset,
		length: size,
	},
	}
	t.self = t
	return t
}

type genericTable struct {
	TableBase
}

func (t *genericTable) Base() *TableBase {
	return &t.TableBase
}

// TableBase is a common parent for all kinds of OpenType tables.
type TableBase struct {
	data   fontBinSegm // a table is a slice of font data
	name   Tag         // 4-byte name as an integer
	offset uint32      // from offset
	length uint32      // to offset + length
	self   interface{}
}

// Offset returns the offset of this table within the OpenType font.
func (tb *TableBase) Offset() uint32 {
	return tb.offset
}

// Len returns the size of this table in bytes.
func (tb *TableBase) Len() uint32 {
	return tb.length
}

// Binary returns the bytes of this table. Should be treatet as read-only by
// clients, as it is a view into the original data.
func (tb *TableBase) Binary() []byte {
	return tb.data
}

// String returns the 4-letter name of a table.
func (tb *TableBase) String() string {
	return tb.name.String()
}

func (tb *TableBase) bytes() fontBinSegm {
	return tb.data
}

// AsGPos returns this table as a GPOS table, or nil.
func (tb *TableBase) AsGPos() *GPosTable {
	if tb == nil || tb.self == nil {
		return nil
	}
	if g, ok := tb.self.(*GPosTable); ok {
		return g
	}
	return nil
}

// AsGSub returns this table as a GSUB table, or nil.
func (tb *TableBase) AsGSub() *GSubTable {
	if tb == nil || tb.self == nil {
		return nil
	}
	if g, ok := tb.self.(*GSubTable); ok {
		return g
	}
	return nil
}

// AsLoca returns this table as a kern table, or nil.
func (tb *TableBase) AsLoca() *LocaTable {
	if tb == nil || tb.self == nil {
		return nil
	}
	if k, ok := tb.self.(*LocaTable); ok {
		return k
	}
	return nil
}

// AsMaxP returns this table as a kern table, or nil.
func (tb *TableBase) AsMaxP() *MaxPTable {
	if tb == nil || tb.self == nil {
		return nil
	}
	if k, ok := tb.self.(*MaxPTable); ok {
		return k
	}
	return nil
}

// AsKern returns this table as a kern table, or nil.
func (tb *TableBase) AsKern() *KernTable {
	if tb == nil || tb.self == nil {
		return nil
	}
	if k, ok := tb.self.(*KernTable); ok {
		return k
	}
	return nil
}

// AsHHea returns this table as a hhea table, or nil.
func (tb *TableBase) AsHHea() *HHeaTable {
	if tb == nil || tb.self == nil {
		return nil
	}
	if k, ok := tb.self.(*HHeaTable); ok {
		return k
	}
	return nil
}

// AsHMtx returns this table as a hhea table, or nil.
func (tb *TableBase) AsHMtx() *HMtxTable {
	if tb == nil || tb.self == nil {
		return nil
	}
	if k, ok := tb.self.(*HMtxTable); ok {
		return k
	}
	return nil
}

// --- Concrete table implementations ----------------------------------------

// HeadTable gives global information about the font.
type HeadTable struct {
	TableBase
	Flags            uint16
	UnitsPerEm       uint16
	IndexToLocFormat uint16 // needed to read loca table
}

func newHeadTable(tag Tag, b fontBinSegm, offset, size uint32) *HeadTable {
	t := &HeadTable{}
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

func (t *HeadTable) Base() *TableBase {
	return &t.TableBase
}

// KernTable gives information about kerning and kern pairs.
// The kerning table contains the values that control the inter-character spacing for
// the glyphs in a font. OpenType™ fonts containing CFF outlines are not supported
// by the 'kern' table and must use the GPOS OpenType Layout table.
type KernTable struct {
	TableBase
	headers []kernSubTableHeader
}

func newKernTable(tag Tag, b fontBinSegm, offset, size uint32) *KernTable {
	t := &KernTable{}
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

// KernSubTableInfo contains header information for a kerning sub-table.
// Currently only format 0 of kerning tables is supported (as does MS Windows).
type KernSubTableInfo struct {
	IsHorizontal  bool // kern data may be horizontal or vertical
	IsMinimum     bool // if false, table has kerning values, otherwise has minimum values
	IsOverride    bool // if true, the value in this table should replace the value currently being accumulated
	IsCrossStream bool // if true, kerning is perpendicular to the flow of the text
	Offset        uint16
	Length        uint32
}

// SubTableInfo returns information about a kerning sub-table. n is 0…N-1.
func (t *KernTable) SubTableInfo(n int) KernSubTableInfo {
	// Mask    Name
	// 0x8000  kernVertical
	// 0x4000  kernCrossStream
	// 0x2000  kernVariation
	// 0x1000  kernOverride
	// 0x0F00  kernUnusedBits
	// 0x00FF  kernFormatMask
	info := KernSubTableInfo{}
	if len(t.headers) >= n {
		h := t.headers[n]
		info.IsHorizontal = h.coverage&0x8000 == 0
		info.IsMinimum = h.coverage&0x4000 > 0
		info.IsCrossStream = h.coverage&0x2000 > 0
		info.IsOverride = h.coverage&0x08 > 0
		info.Offset = h.offset
		info.Length = h.length
	}
	return info
}

func (t *KernTable) Base() *TableBase {
	return &t.TableBase
}

// LocaTable stores the offsets to the locations of the glyphs in the font,
// relative to the beginning of the glyph data table.
// By definition, index zero points to the “missing character”, which is the character
// that appears if a character is not found in the font. The missing character is
// commonly represented by a blank box or a space.
type LocaTable struct {
	TableBase
	loca func(t *LocaTable, n int) uint32 // returns glyph location for glyph n
}

func newLocaTable(tag Tag, b fontBinSegm, offset, size uint32) *LocaTable {
	t := &LocaTable{}
	base := TableBase{
		data:   b,
		name:   tag,
		offset: offset,
		length: size,
	}
	t.TableBase = base
	t.loca = shortLocaVersion // may get changed by font consistency check
	t.self = t
	return t
}

func shortLocaVersion(t *LocaTable, n int) uint32 {
	loc, err := t.data.u16(n * 2)
	if err != nil {
		// should have been catched by font consistency check
		panic("access to non-existent loca offset")
	}
	return uint32(loc) * 2
}

func longLocaVersion(t *LocaTable, n int) uint32 {
	loc, err := t.data.u32(n * 4)
	if err != nil {
		// should have been catched by font consistency check
		panic("access to non-existent loca offset")
	}
	return loc
}

func (t *LocaTable) Base() *TableBase {
	return &t.TableBase
}

// MaxPTable establishes the memory requirements for this font.
// The 'maxp' table contains a count for the number of glyphs in the font.
// Whenever this value changes, other tables which depend on it should also be updated.
type MaxPTable struct {
	TableBase
	NumGlyphs int
}

func newMaxPTable(tag Tag, b fontBinSegm, offset, size uint32) *MaxPTable {
	t := &MaxPTable{}
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

func (t *MaxPTable) Base() *TableBase {
	return &t.TableBase
}

// HHeaTable contains information for horizontal layout.
type HHeaTable struct {
	TableBase
	NumberOfHMetrics int
}

func newHHeaTable(tag Tag, b fontBinSegm, offset, size uint32) *HHeaTable {
	t := &HHeaTable{}
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

func (t *HHeaTable) Base() *TableBase {
	return &t.TableBase
}

// HMtxTable contains metric information for the horizontal layout each of the glyphs in
// the font. Each element in the contained hMetrics-array has two parts: the advance width
// and left side bearing. The value NumberOfHMetrics is taken from the `hhea` table. In
// a monospaced font, only one entry is required but that entry may not be omitted.
// Optionally, an array of left side bearings follows.
// The corresponding glyphs are assumed to have the same
// advance width as that found in the last entry in the hMetrics array. Since there
// must be a left side bearing and an advance width associated with each glyph in the font,
// the number of entries in this array is derived from the total number of glyphs in the
// font minus the value `HHea.NumberOfHMetrics`, which is copied into the
// HMtxTable for easier access.
type HMtxTable struct {
	TableBase
	NumberOfHMetrics int
}

func newHMtxTable(tag Tag, b fontBinSegm, offset, size uint32) *HMtxTable {
	t := &HMtxTable{}
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

// hMetrics returns the advance width and left side bearing of a glyph.
// TODO: call from font or from HMtx ?
func (t *HMtxTable) hMetrics(g GlyphIndex) (uint16, int16) {
	if t.NumberOfHMetrics < int(g) {
		a, _ := t.data.u16(int(g) * 4)
		lsb, _ := t.data.u16(int(g)*4 + 2)
		return a, int16(lsb)
	}
	diff := int(g) - t.NumberOfHMetrics
	a, _ := t.data.u16((t.NumberOfHMetrics - 1) * 4)
	lsb, _ := t.data.u16((t.NumberOfHMetrics-1)*4 + diff*2)
	return a, int16(lsb)
}

func (t *HMtxTable) Base() *TableBase {
	return &t.TableBase
}
