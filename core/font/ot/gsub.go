package ot

import (
	"strconv"
)

// GSubTable is a type representing an OpenType GSUB table
// (see https://docs.microsoft.com/en-us/typography/opentype/spec/gpos).
type GSubTable struct {
	LayoutTable
}

func newGSubTable(tag Tag, b fontBinSegm, offset, size uint32) *GSubTable {
	t := &GSubTable{}
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

// Base returns the enclosed LayoutTable type this table inherits from.
func (g *GSubTable) LayoutBase() *LayoutTable {
	return &g.LayoutTable
}

// Base returns the enclosed TableBase type this table inherits from.
func (g *GSubTable) Base() *TableBase {
	return &g.TableBase
}

var _ Table = &GSubTable{}

// --- GSUB lookup record types ----------------------------------------------

type lookupSubstFormat1 struct {
	Format         uint16 // 1
	CoverageOffset uint16 // Offset to Coverage table, from beginning of substitution subtable
	Glyphs         int16  // Changes meaning, depending on format 1 or 2
}

type lookupSubstFormat2 struct {
	lookupSubstFormat1
	SubstituteGlyphIDs []uint16
}

// GSUB Table Lookup Type
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#table-organization

// GSUB Lookup Type Enumeration
const (
	GSUB_LUTYPE_Single           LayoutTableLookupType = 1 // Replace one glyph with one glyph
	GSUB_LUTYPE_Multiple         LayoutTableLookupType = 2 // Replace one glyph with more than one glyph
	GSUB_LUTYPE_Alternate        LayoutTableLookupType = 3 // Replace one glyph with one of many glyphs
	GSUB_LUTYPE_Ligature         LayoutTableLookupType = 4 // Replace multiple glyphs with one glyph
	GSUB_LUTYPE_Context          LayoutTableLookupType = 5 // Replace one or more glyphs in context
	GSUB_LUTYPE_Chaining_Context LayoutTableLookupType = 6 // Replace one or more glyphs in chained context
	GSUB_LUTYPE_Extension_Subs   LayoutTableLookupType = 7 // Extension mechanism for other substitutions
	GSUB_LUTYPE_Reverse_Chaining LayoutTableLookupType = 8 // Applied in reverse order, replace single glyph in chaining context
)

const gsubLookupTypeNames = "Single|Multiple|Ligature|Alternate|Context|Chaining|Extension|Reverse"

var gsubLookupTypeInx = [...]int{0, 7, 16, 25, 35, 43, 52, 62, 70}

// GSubString interprets a layout table lookup type as a GSUB table type.
func (lt LayoutTableLookupType) GSubString() string {
	lt -= 1
	if lt >= 0 && lt < GSUB_LUTYPE_Reverse_Chaining {
		return gsubLookupTypeNames[gsubLookupTypeInx[lt] : gsubLookupTypeInx[lt+1]-1]
	}
	return strconv.Itoa(int(lt))
}
