package ot

import "strconv"

// GPosTable is a type representing an OpenType GPOS table
// (see https://docs.microsoft.com/en-us/typography/opentype/spec/gsub).
type GPosTable struct {
	tableBase
	LayoutTable
}

func newGPosTable(tag Tag, b binarySegm, offset, size uint32) *GPosTable {
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
	GPosLookupTypeSingle            LayoutTableLookupType = 1 // Adjust position of a single glyph
	GPosLookupTypePair              LayoutTableLookupType = 2 // Adjust position of a pair of glyphs
	GPosLookupTypeCursive           LayoutTableLookupType = 3 // Attach cursive glyphs
	GPosLookupTypeMarkToBase        LayoutTableLookupType = 4 // Attach a combining mark to a base glyph
	GPosLookupTypeMarkToLigature    LayoutTableLookupType = 5 // Attach a combining mark to a ligature
	GPosLookupTypeMarkToMark        LayoutTableLookupType = 6 // Attach a combining mark to another mark
	GPosLookupTypeContextPos        LayoutTableLookupType = 7 // Position one or more glyphs in context
	GPosLookupTypeChainedContextPos LayoutTableLookupType = 8 // Position one or more glyphs in chained context
	GPosLookupTypeExtensionPos      LayoutTableLookupType = 9 // Extension mechanism for other positionings
)

const gposLookupTypeNames = "Single|Pair|Cursive|MarkToBase|MarkToLigature|MarkToMark|ContextPos|Chained|Ext"

var gposLookupTypeInx = [...]int{0, 7, 12, 20, 31, 46, 57, 68, 76, 80}

// GPosString interprets a layout table lookup type as a GPOS table type.
func (lt LayoutTableLookupType) GPosString() string {
	lt -= 1
	if lt < GPosLookupTypeExtensionPos {
		return gposLookupTypeNames[gposLookupTypeInx[lt] : gposLookupTypeInx[lt+1]-1]
	}
	return strconv.Itoa(int(lt))
}
