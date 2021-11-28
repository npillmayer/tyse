package ot

import (
	"strconv"
)

// GSubTable is a type representing an OpenType GSUB table
// (see https://docs.microsoft.com/en-us/typography/opentype/spec/gpos).
type GSubTable struct {
	tableBase
	LayoutTable
}

func newGSubTable(tag Tag, b binarySegm, offset, size uint32) *GSubTable {
	t := &GSubTable{}
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
	GSubLookupTypeSingle          LayoutTableLookupType = 1 // Replace one glyph with one glyph
	GSubLookupTypeMultiple        LayoutTableLookupType = 2 // Replace one glyph with more than one glyph
	GSubLookupTypeAlternate       LayoutTableLookupType = 3 // Replace one glyph with one of many glyphs
	GSubLookupTypeLigature        LayoutTableLookupType = 4 // Replace multiple glyphs with one glyph
	GSubLookupTypeContext         LayoutTableLookupType = 5 // Replace one or more glyphs in context
	GSubLookupTypeChainingContext LayoutTableLookupType = 6 // Replace one or more glyphs in chained context
	GSubLookupTypeExtensionSubs   LayoutTableLookupType = 7 // Extension mechanism for other substitutions
	GSubLookupTypeReverseChaining LayoutTableLookupType = 8 // Applied in reverse order, replace single glyph in chaining context
)

const gsubLookupTypeNames = "Single|Multiple|Alternate|Ligature|Context|Chaining|Extension|Reverse"

var gsubLookupTypeInx = [...]int{0, 7, 16, 26, 35, 43, 52, 62, 70}

// GSubString interprets a layout table lookup type as a GSUB table type.
func (lt LayoutTableLookupType) GSubString() string {
	lt -= 1
	if lt < GSubLookupTypeReverseChaining {
		return gsubLookupTypeNames[gsubLookupTypeInx[lt] : gsubLookupTypeInx[lt+1]-1]
	}
	return strconv.Itoa(int(lt))
}
