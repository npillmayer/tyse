package ot

import "strconv"

// GPosTable is a type representing an OpenType GPOS table
// (see https://docs.microsoft.com/en-us/typography/opentype/spec/gsub).
type GPosTable struct {
	tableBase
	LayoutTable
}

func newGPosTable(tag Tag, b fontBinSegm, offset, size uint32) *GPosTable {
	t := &GPosTable{}
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

var _ Table = &GPosTable{}

// GPOS Table
// https://docs.microsoft.com/en-us/typography/opentype/spec/gpos#table-organization

// GPOS Lookup Type Enumeration
const (
	GPOS_LUTYPE_Single            LayoutTableLookupType = 1 // Adjust position of a single glyph
	GPOS_LUTYPE_Pair              LayoutTableLookupType = 2 // Adjust position of a pair of glyphs
	GPOS_LUTYPE_Cursive           LayoutTableLookupType = 3 // Attach cursive glyphs
	GPOS_LUTYPE_MarkToBase        LayoutTableLookupType = 4 // Attach a combining mark to a base glyph
	GPOS_LUTYPE_MarkToLigature    LayoutTableLookupType = 5 // Attach a combining mark to a ligature
	GPOS_LUTYPE_MarkToMark        LayoutTableLookupType = 6 // Attach a combining mark to another mark
	GPOS_LUTYPE_ContextPos        LayoutTableLookupType = 7 // Position one or more glyphs in context
	GPOS_LUTYPE_ChainedContextPos LayoutTableLookupType = 8 // Position one or more glyphs in chained context
	GPOS_LUTYPE_ExtensionPos      LayoutTableLookupType = 9 // Extension mechanism for other positionings
)

const xxxxLookupTypeNames = "0123456789 123456789 123456789 123456789 123456789 123456789 123456789 123456789"
const gposLookupTypeNames = "Single|Pair|Cursive|MarkToBase|MarkToLigature|MarkToMark|ContextPos|Chained|Ext"

var gposLookupTypeInx = [...]int{0, 7, 12, 20, 31, 46, 57, 68, 76, 80}

// GPosString interprets a layout table lookup type as a GPOS table type.
func (lt LayoutTableLookupType) GPosString() string {
	lt -= 1
	if lt >= 0 && lt < GPOS_LUTYPE_ExtensionPos {
		return gposLookupTypeNames[gposLookupTypeInx[lt] : gposLookupTypeInx[lt+1]-1]
	}
	return strconv.Itoa(int(lt))
}
