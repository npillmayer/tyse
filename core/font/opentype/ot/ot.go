package ot

import (
	"fmt"

	"github.com/npillmayer/tyse/core/font"
	"golang.org/x/text/encoding/unicode"
)

// Font represents the internal structure of an OpenType font.
// It is used to navigate properties of a font for typesetting tasks.
//
// We only support OpenType fonts with advanced layout, i.e. fonts containing tables
// GSUB, GPOS, etc.
type Font struct {
	F      *font.ScalableFont
	Header *FontHeader
	tables map[Tag]Table
	CMap   *CMapTable // CMAP table is mandatory
	Layout struct {   // OpenType core layout tables
		GSub *GSubTable // OpenType layout GSUB
		GPos *GPosTable // OpenType layout GPOS
		GDef *GDefTable // OpenType layout GDEF
		Base *BaseTable // OpenType layout BASE
		// TODO JSTF
	}
}

// FontHeader is a directory of the top-level tables in a font. If the font file
// contains only one font, the table directory will begin at byte 0 of the file.
// If the font file is an OpenType Font Collection file (see below), the beginning
// point of the table directory for each font is indicated in the TTCHeader.
//
// OpenType fonts that contain TrueType outlines should use the value of 0x00010000
// for the FontType. OpenType fonts containing CFF data (version 1 or 2) should
// use 0x4F54544F ('OTTO', when re-interpreted as a Tag).
// The Apple specification for TrueType fonts allows for 'true' and 'typ1',
// but these version tags should not be used for OpenType fonts.
type FontHeader struct {
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
//	os2  := otf.Table(ot.T("OS/2"))
//	loca := otf.Table(ot.T("loca")).Self().AsLoca()
//
// Table tag names are case-sensitive, following the names in the OpenType specification,
// i.e., one of:
//
// avar BASE CBDT CBLC CFF CFF2 cmap COLR CPAL cvar cvt DSIG EBDT EBLC EBSC fpgm fvar
// gasp GDEF glyf GPOS GSUB gvar hdmx head hhea hmtx HVAR JSTF kern loca LTSH MATH
// maxp MERG meta MVAR name OS/2 PCLT post prep sbix STAT SVG VDMX vhea vmtx VORG VVAR
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

// --- Tag -------------------------------------------------------------------

// Tag is defined by the spec as:
// Array of four uint8s (length = 32 bits) used to identify a table, design-variation axis,
// script, language system, feature, or baseline
type Tag uint32

// MakeTag creates a Tag from 4 bytes, e.g.,
// If b is shorter or longer, it will be silently extended or cut as appropriate
//
//	MakeTag([]byte("cmap"))
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
	t = (t + "    ")[:4]
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
type Table interface {
	Extent() (uint32, uint32) // offset and byte size within the font's binary data
	Binary() []byte           // the bytes of this table; should be treatet as read-only by clients
	Fields() Navigator        // start for navigation calls
	Self() TableSelf          // reference to itself
}

func newTable(tag Tag, b binarySegm, offset, size uint32) *genericTable {
	t := &genericTable{tableBase{
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
	tableBase
}

// tableBase is a common parent for all kinds of OpenType tables.
type tableBase struct {
	data   binarySegm // a table is a slice of font data
	name   Tag        // 4-byte name as an integer
	offset uint32     // from offset
	length uint32     // to offset + length
	self   interface{}
}

// Offset returns offset and byte size of this table within the OpenType font.
func (tb *tableBase) Extent() (uint32, uint32) {
	return tb.offset, tb.length
}

// Binary returns the bytes of this table. Should be treatet as read-only by
// clients, as it is a view into the original data.
func (tb *tableBase) Binary() []byte {
	return tb.data
}

// func (tb *tableBase) bytes() fontBinSegm {
// 	return tb.data
// }

func (tb *tableBase) Self() TableSelf {
	return TableSelf{tableBase: tb}
}

func (tb *tableBase) Fields() Navigator {
	tableTag := tb.name.String()
	return NavigatorFactory(tableTag, tb.data, tb.data)
}

// TableSelf is a reference to a table. Its primary use is for converting
// a generic table to a concrete table flavour, and for reproducing the
// name tag of a table.
type TableSelf struct {
	tableBase *tableBase
}

// NameTag returns the 4-letter name of a table.
func (tself TableSelf) NameTag() Tag {
	return tself.tableBase.name
}

func safeSelf(tself TableSelf) interface{} {
	if tself.tableBase == nil || tself.tableBase.self == nil {
		return TableSelf{}
	}
	return tself.tableBase.self
}

// AsCMap returns this table as a cmap table, or nil.
func (tself TableSelf) AsCMap() *CMapTable {
	if k, ok := safeSelf(tself).(*CMapTable); ok {
		return k
	}
	return nil
}

// AsGPos returns this table as a GPOS table, or nil.
func (tself TableSelf) AsGPos() *GPosTable {
	if g, ok := safeSelf(tself).(*GPosTable); ok {
		return g
	}
	return nil
}

// AsGSub returns this table as a GSUB table, or nil.
func (tself TableSelf) AsGSub() *GSubTable {
	if g, ok := safeSelf(tself).(*GSubTable); ok {
		return g
	}
	return nil
}

// AsGDef returns this table as a GDEF table, or nil.
func (tself TableSelf) AsGDef() *GDefTable {
	if g, ok := safeSelf(tself).(*GDefTable); ok {
		return g
	}
	return nil
}

// AsBase returns this table as a BASE table, or nil.
func (tself TableSelf) AsBase() *BaseTable {
	if k, ok := safeSelf(tself).(*BaseTable); ok {
		return k
	}
	return nil
}

// AsLoca returns this table as a kern table, or nil.
func (tself TableSelf) AsLoca() *LocaTable {
	if k, ok := safeSelf(tself).(*LocaTable); ok {
		return k
	}
	return nil
}

// AsMaxP returns this table as a kern table, or nil.
func (tself TableSelf) AsMaxP() *MaxPTable {
	if k, ok := safeSelf(tself).(*MaxPTable); ok {
		return k
	}
	return nil
}

// AsKern returns this table as a kern table, or nil.
func (tself TableSelf) AsKern() *KernTable {
	if k, ok := safeSelf(tself).(*KernTable); ok {
		return k
	}
	return nil
}

// AsHead returns this table as a head table, or nil.
func (tself TableSelf) AsHead() *HeadTable {
	if k, ok := safeSelf(tself).(*HeadTable); ok {
		return k
	}
	return nil
}

// AsHHea returns this table as a hhea table, or nil.
func (tself TableSelf) AsHHea() *HHeaTable {
	if k, ok := safeSelf(tself).(*HHeaTable); ok {
		return k
	}
	return nil
}

// AsHMtx returns this table as a hmtx table, or nil.
func (tself TableSelf) AsHMtx() *HMtxTable {
	if k, ok := safeSelf(tself).(*HMtxTable); ok {
		return k
	}
	return nil
}

// --- Concrete table implementations ----------------------------------------

// HeadTable gives global information about the font.
// Only a small subset of fields are made public by HeadTable, as they are
// needed for consistency-checks. To read any of the other fields of table 'head' use:
//
//	head   := otf.Table(T("head"))
//	fields := head.Fields().Get(n)     // get nth field value
//	fields := head.Fields().All()      // get a slice with all field values
//
// See also type `Navigator`.
type HeadTable struct {
	tableBase
	Flags            uint16 // see https://docs.microsoft.com/en-us/typography/opentype/spec/head
	UnitsPerEm       uint16 // values 16 … 16384 are valid
	IndexToLocFormat uint16 // needed to interpret loca table
}

func newHeadTable(tag Tag, b binarySegm, offset, size uint32) *HeadTable {
	t := &HeadTable{}
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

// KernTable gives information about kerning and kern pairs.
// The kerning table contains the values that control the inter-character spacing for
// the glyphs in a font. OpenType™ fonts containing CFF outlines are not supported
// by the 'kern' table and must use the GPOS OpenType Layout table.
type KernTable struct {
	tableBase
	headers []kernSubTableHeader
}

func newKernTable(tag Tag, b binarySegm, offset, size uint32) *KernTable {
	t := &KernTable{}
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

// LocaTable stores the offsets to the locations of the glyphs in the font,
// relative to the beginning of the glyph data table.
// By definition, index zero points to the “missing character”, which is the character
// that appears if a character is not found in the font. The missing character is
// commonly represented by a blank box or a space.
type LocaTable struct {
	tableBase
	inx2loc func(t *LocaTable, gid GlyphIndex) uint32 // returns glyph location for glyph gid
	locCnt  int                                       // number of locations
}

// IndexToLocation offsets, indexed by glyph IDs, which provide the location of each
// glyph data block within the 'glyf' table.
func (t *LocaTable) IndexToLocation(gid GlyphIndex) uint32 {
	return t.inx2loc(t, gid)
}

func newLocaTable(tag Tag, b binarySegm, offset, size uint32) *LocaTable {
	t := &LocaTable{}
	base := tableBase{
		data:   b,
		name:   tag,
		offset: offset,
		length: size,
	}
	t.tableBase = base
	t.inx2loc = shortLocaVersion // may get changed by font consistency check
	t.locCnt = 0                 // has to be set during consistency check
	t.self = t
	return t
}

func shortLocaVersion(t *LocaTable, gid GlyphIndex) uint32 {
	// in case of error link to 'missing character' at location 0
	if gid >= GlyphIndex(t.locCnt) {
		return 0
	}
	loc, err := t.data.u16(int(gid) * 2)
	if err != nil {
		return 0
	}
	return uint32(loc) * 2
}

func longLocaVersion(t *LocaTable, gid GlyphIndex) uint32 {
	// in case of error link to 'missing character' at location 0
	if gid >= GlyphIndex(t.locCnt) {
		return 0
	}
	loc, err := t.data.u32(int(gid) * 4)
	if err != nil {
		return 0
	}
	return loc
}

// MaxPTable establishes the memory requirements for this font.
// The 'maxp' table contains a count for the number of glyphs in the font.
// Whenever this value changes, other tables which depend on it should also be updated.
type MaxPTable struct {
	tableBase
	NumGlyphs int
}

func newMaxPTable(tag Tag, b binarySegm, offset, size uint32) *MaxPTable {
	t := &MaxPTable{}
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

// HHeaTable contains information for horizontal layout.
type HHeaTable struct {
	tableBase
	NumberOfHMetrics int
}

func newHHeaTable(tag Tag, b binarySegm, offset, size uint32) *HHeaTable {
	t := &HHeaTable{}
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
	tableBase
	NumberOfHMetrics int
}

func newHMtxTable(tag Tag, b binarySegm, offset, size uint32) *HMtxTable {
	t := &HMtxTable{}
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

// Names struct for table 'name'
type nameNames struct {
	navBase
	strbuf   binarySegm
	nameRecs array
}

func (n nameNames) Name() string {
	return "name" // name of 'name' OT table
}

func (n nameNames) Map() NavMap {
	namesMap := make(map[Tag]link16)
	for i := 0; i < n.nameRecs.length; i++ {
		nameRecord := n.nameRecs.Get(i)
		pltf := nameRecord.U16(0)
		enc := nameRecord.U16(2)
		if !((pltf == 0 && enc == 3) || (pltf == 3 && enc == 1)) {
			//trace().Debugf("unsupported platform/encoding combination for name-table")
			continue
		}
		id := nameRecord.U16(6)
		strlen := nameRecord.U16(8)
		offset := nameRecord.U16(10)
		str := n.strbuf[offset : offset+strlen] // UTF-16 encoded string
		//trace().Debugf("utf16 string = '%v'", decodeUtf16(str))
		link := makeLink16(0, str, "NameRecord")
		tag := MakeTag([]byte{byte(pltf), byte(enc), 0, byte(id)})
		//trace().Debugf("copying names[0x%x] = %d", tag, nameRecord)
		namesMap[tag] = link.(link16)
	}
	return mapWrapper{m: namesMap, names: n, name: n.Name()}
}

func decodeUtf16(str []byte) (string, error) {
	enc := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
	decoder := enc.NewDecoder()
	s, err := decoder.Bytes(str)
	if err != nil {
		return "", fmt.Errorf("decoding UTF-16 error: %v", err)
	}
	return string(s), nil
}
