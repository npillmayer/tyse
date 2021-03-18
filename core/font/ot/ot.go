package ot

import (
	"github.com/ConradIrwin/font/sfnt"
	"github.com/npillmayer/tyse/core/font"
)

// OTFont represents the internal structure of an OpenType font.
// It is used to navigate properties of a font for typesetting tasks.
type OTFont struct {
	f          *font.ScalableFont
	ot         *sfnt.Font // TODO remove this and dependency to ConradIrwin/font
	header     *fontHeader
	tables     map[Tag]Table
	glyphIndex glyphIndexFunc
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
func (otf *OTFont) GlyphIndex(codePoint rune) (GlyphIndex, error) {
	return otf.glyphIndex(otf, codePoint)
}

// --- Constants -------------------------------------------------------------

// LayoutTableSectionName lists the sections of OT layout tables, i.e. GPOS and GSUB.
type LayoutTableSectionName int

const (
	LayoutScriptSection LayoutTableSectionName = iota
	LayoutFeatureSection
	LayoutLookupSection
	LayoutFeatureVariationsSection
)

// Lookup Tables
// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#lookup-table

// Lookup flags of layout tables (GPOS and GSUB)
const ( // LookupFlag bit enumeration
	// Note that the RIGHT_TO_LEFT flag is used only for GPOS type 3 lookups and is ignored
	// otherwise. It is not used by client software in determining text direction.
	LOOKUP_FLAG_RIGHT_TO_LEFT             uint16 = 0x0001
	LOOKUP_FLAG_IGNORE_BASE_GLYPHS        uint16 = 0x0002 // If set, skips over base glyphs
	LOOKUP_FLAG_IGNORE_LIGATURES          uint16 = 0x0004 // If set, skips over ligatures
	LOOKUP_FLAG_IGNORE_MARKS              uint16 = 0x0008 // If set, skips over all combining marks
	LOOKUP_FLAG_USE_MARK_FILTERING_SET    uint16 = 0x0010 // If set, indicates that the lookup table structure is followed by a MarkFilteringSet field.
	LOOKUP_FLAG_reserved                  uint16 = 0x00E0 // For future use (Set to zero)
	LOOKUP_FLAG_MARK_ATTACHMENT_TYPE_MASK uint16 = 0xFF00 // If not zero, skips over all marks of attachment type different from specified.
)

// GPOS Table
// https://docs.microsoft.com/en-us/typography/opentype/spec/gpos#table-organization

// GPOS LookupType Enumeration
const (
	GPOS_LUTYPE_Single              uint16 = 1 // Adjust position of a single glyph
	GPOS_LUTYPE_Pair                uint16 = 2 // Adjust position of a pair of glyphs
	GPOS_LUTYPE_Cursive             uint16 = 3 // Attach cursive glyphs
	GPOS_LUTYPE_MarkToBase          uint16 = 4 // Attach a combining mark to a base glyph
	GPOS_LUTYPE_MarkToLigature      uint16 = 5 // Attach a combining mark to a ligature
	GPOS_LUTYPE_MarkToMark          uint16 = 6 // Attach a combining mark to another mark
	GPOS_LUTYPE_Context_Pos         uint16 = 7 // Position one or more glyphs in context
	GPOS_LUTYPE_Chained_Context_Pos uint16 = 8 // Position one or more glyphs in chained context
	GPOS_LUTYPE_Extension_Pos       uint16 = 9 // Extension mechanism for other positionings
)

// --- Tag -------------------------------------------------------------------

// Tag is defined by the spec as:
// Array of four uint8s (length = 32 bits) used to identify a table, design-variation axis,
// script, language system, feature, or baseline
type Tag uint32

// tag creates a tag from 4 bytes.
func tag(b []byte) Tag {
	return Tag(u32(b))
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
type Table interface {
	Offset() uint32   // offset within the font's binary data
	Len() uint32      // byte size of table
	String() string   // 4-letter table name, e.g., "cmap"
	Base() *TableBase // every table we use will be derived from TableBase
}

func newTable(tag Tag, b fontBinSegm, offset, size uint32) Table {
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

// String returns the 4-letter name of a table.
func (tb *TableBase) String() string {
	return tb.name.String()
}

func (tb *TableBase) bytes() fontBinSegm {
	return tb.data
}

// AsGSub returns this table as a GSUB table, or nil.
func (tb *TableBase) AsGSub() *GSubTable {
	if tb.self == nil {
		return nil
	}
	if g, ok := tb.self.(*GSubTable); ok {
		return g
	}
	return nil
}

// HeadTable gives global information about the font.
type HeadTable struct {
	TableBase
	flags      uint16
	unitsPerEm uint16
}

func (t *HeadTable) Base() *TableBase {
	return &t.TableBase
}

// --- Font header -----------------------------------------------------------

type fontHeader struct {
	FontType      uint32
	TableCount    uint16
	EntrySelector uint16
	SearchRange   uint16
	RangeShift    uint16
}
