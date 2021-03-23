package ot

/*
From https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2:

OpenType Layout consists of five tables: the Glyph Substitution table (GSUB),
the Glyph Positioning table (GPOS), the Baseline table (BASE),
the Justification table (JSTF), and the Glyph Definition table (GDEF).
These tables use some of the same data formats.
*/

// --- Layout tables ---------------------------------------------------------

// LayoutTable is a base type for layout tables.
// OpenType specifies two tables–GPOS and GSUB–which share some of their
// structure. They are called "layout tables".
type LayoutTable struct {
	TableBase
	Scripts  TagRecordMap
	Features TagRecordMap
	lookups  []*lookupRecord
	header   *LayoutHeader
}

// Header returns the layout table header for this GSUB table.
func (t *LayoutTable) Header() LayoutHeader {
	return *t.header
}

// LayoutHeader represents header information common to the layout tables.
type LayoutHeader struct {
	versionHeader
	offsets layoutHeader11
}

// Version returns major and minor version numbers for this layout table.
func (h LayoutHeader) Version() (int, int) {
	return int(h.Major), int(h.Minor)
}

// OffsetFor returns an offset for a layout table section within the layout table
// (GPOS or GSUB).
// A layout table contains four sections:
// ▪︎ Script Section,
// ▪︎ Feature Section,
// ▪︎ Lookup Section,
// ▪︎ Feature Variations Section.
// (see type LayoutTableSectionName)
//
func (h *LayoutHeader) OffsetFor(which LayoutTableSectionName) int {
	switch which {
	case LayoutScriptSection:
		return int(h.offsets.ScriptListOffset)
	case LayoutFeatureSection:
		return int(h.offsets.FeatureListOffset)
	case LayoutLookupSection:
		return int(h.offsets.LookupListOffset)
	case LayoutFeatureVariationsSection:
		return int(h.offsets.FeatureVariationsOffset)
	}
	trace().Errorf("illegal section offset type into layout table: %d", which)
	return 0 // illegal call, nothing sensible to return
}

// versionHeader is the beginning of on-disk format of some format headers.
// See https://docs.microsoft.com/en-us/typography/opentype/spec/gdef#gdef-header
// See https://www.microsoft.com/typography/otspec/GPOS.htm
// See https://www.microsoft.com/typography/otspec/GSUB.htm
// Fields are public for reflection-access.
type versionHeader struct {
	Major uint16
	Minor uint16
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

// --- GDEF table ------------------------------------------------------------

// GDefTable, the Glyph Definition (GDEF) table, provides various glyph properties
// used in OpenType Layout processing.
//
// See also
// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#class-definition-table
type GDefTable struct {
	TableBase
	header             GDefHeader
	classDef           ClassDefinitions
	attachPointList    AttachmentPointList
	markAttachClassDef ClassDefinitions
	markGlyphSets      []GlyphRange
}

func newGDefTable(tag Tag, b fontBinSegm, offset, size uint32) *GDefTable {
	t := &GDefTable{}
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

func (t *GDefTable) Base() *TableBase {
	return &t.TableBase
}

// Header returns the Glyph Definition header for t.
func (t *GDefTable) Header() GDefHeader {
	return t.header
}

// GDefHeader contains general information for a Glyph Definition table (GDEF).
type GDefHeader struct {
	gDefHeader
}

// Version returns major and minor version numbers for this GDef table.
func (h GDefHeader) Version() (int, int) {
	return int(h.Major), int(h.Minor)
}

// gDefHeader starts with a version number. Three versions are defined:
// 1.0, 1.2 and 1.3.
type gDefHeader struct {
	gDefHeaderV1_0
	MarkGlyphSetsDefOffset uint16
	ItemVarStoreOffset     uint32
	headerSize             uint8 // header size in bytes
}

type gDefHeaderV1_0 struct {
	versionHeader
	GlyphClassDefOffset      uint16
	AttachListOffset         uint16
	LigCaretListOffset       uint16
	MarkAttachClassDefOffset uint16
}

// GDefTableSectionName lists the sections of a GDEF table.
type GDefTableSectionName int

const (
	GDefGlyphClassDefSection GDefTableSectionName = iota
	GDefAttachListSection
	GDefLigCaretListSection
	GDefMarkAttachClassSection
	GDefMarkGlyphSetsDefSection
	GDefItemVarStoreSection
)

// OffsetFor returns an offset for a table section within the GDEF table.
// A GDEF table contains six sections:
// ▪︎ glyph class definitions,
// ▪︎ attachment list definitions,
// ▪︎ ligature carets lists,
// ▪︎ mark attachment class definitions,
// ▪︎ mark glyph sets definitions,
// ▪︎ item variant section.
// (see type GDefTableSectionName)
//
func (h GDefHeader) OffsetFor(which GDefTableSectionName) int {
	switch which {
	case GDefGlyphClassDefSection: // Candidate for a RangeTable
		return int(h.GlyphClassDefOffset)
	case GDefAttachListSection:
		return int(h.AttachListOffset)
	case GDefLigCaretListSection:
		return int(h.LigCaretListOffset)
	case GDefMarkAttachClassSection: // Candidate for a RangeTable
		return int(h.MarkAttachClassDefOffset)
	case GDefMarkGlyphSetsDefSection:
		return int(h.MarkGlyphSetsDefOffset)
	case GDefItemVarStoreSection:
		return int(h.ItemVarStoreOffset)
	}
	trace().Errorf("illegal section offset type into GDEF table: %d", which)
	return 0 // illegal call, nothing sensible to return
}

// --- BASE table ------------------------------------------------------------

// BaseTable, the Baseline table (BASE), provides information used to align glyphs
// of different scripts and sizes in a line of text, whether the glyphs are in the
// same font or in different fonts.
// BaseTable, the Glyph Definition (BSE) table, provides various glyph properties
// used in OpenType Layout processing.
//
// See also
// https://docs.microsoft.com/en-us/typography/opentype/spec/base
type BaseTable struct {
	TableBase
	axisTables [2]AxisTable
}

func newBaseTable(tag Tag, b fontBinSegm, offset, size uint32) *BaseTable {
	t := &BaseTable{}
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

func (t *BaseTable) Base() *TableBase {
	return &t.TableBase
}

type AxisTable struct {
	baselineTags      tagList
	baseScriptRecords TagRecordMap
}

// --- Coverage table module -------------------------------------------------

// Each subtable (except an Extension LookupType subtable) in a lookup references
// a Coverage table (Coverage), which specifies all the glyphs affected by a
// substitution or positioning operation described in the subtable.
// The GSUB, GPOS, and GDEF tables rely on this notion of coverage. If a glyph does
// not appear in a Coverage table, the client can skip that subtable and move
// immediately to the next subtable.

type coverageHeader struct {
	CoverageFormat uint16
	Count          uint16
}

func buildGlyphRangeFromCoverage(chead coverageHeader, b fontBinSegm) GlyphRange {
	if chead.CoverageFormat == 1 {
		return &glyphRangeArray{
			is32:     false,                  // entries are uint16
			count:    int(chead.Count),       // number of entries
			data:     b[4:],                  // header of format 1 coverage table is 4 bytes long
			byteSize: int(4 + chead.Count*2), // header is 4, entries are 2 bytes
		}
	}
	return &glyphRangeRecords{
		is32:     false,                  // entries are uint16
		count:    int(chead.Count),       // number of records
		data:     b[4:],                  // header of format 2 coverage table is 4 bytes long
		byteSize: int(4 + chead.Count*6), // header is 4, entries are 6 bytes
	}
}

// --- Class definition tables -----------------------------------------------

// GlyphClassDefEnum lists the glyph classes for the ClassDefinitions
// ('GlyphClassDef'-table).
type GlyphClassDefEnum uint16

const (
	BaseGlyph      GlyphClassDefEnum = iota //single character, spacing glyph
	LigatureGlyph                           //multiple character, spacing glyph
	MarkGlyph                               //non-spacing combining glyph
	ComponentGlyph                          //part of single character, spacing glyph
)

type ClassDefinitions struct {
	format uint16 // format version 1 or 2
	count  int    // number of entries
	size   uint32 // size in bytes, including header
}

func (cdef *ClassDefinitions) calcSize(numEntries int) uint32 {
	return 0
}

// --- LangSys table ---------------------------------------------------------

type langSys struct {
	err            error
	mandatory      uint16 // 0xffff if unused
	featureIndices array  // list of uint16 indices
}

func (lsys langSys) Link() Link {
	return nullLink("LangSys records not linkable")
}

func (lsys langSys) Map() TagRecordMap {
	return tagRecordMap16{}
}

// entry 0 will be the mandatory feature
func (lsys langSys) List() []uint16 {
	r := make([]uint16, lsys.featureIndices.length+1)
	r[0] = lsys.mandatory
	for i := 0; i < lsys.featureIndices.length; i++ {
		if i < 0 || (i+1)*lsys.featureIndices.recordSize > len(lsys.featureIndices.loc.Bytes()) {
			i = 0
		}
		b, _ := lsys.featureIndices.loc.view(i*lsys.featureIndices.recordSize, lsys.featureIndices.recordSize)
		r[i+1] = u16(b)
	}
	return r
}

func (lsys langSys) IsVoid() bool {
	return lsys.featureIndices.length == 0
}

func (lsys langSys) Error() error {
	return lsys.err
}

func (lsys langSys) Name() string {
	return "LangSys"
}

var _ Navigator = langSys{}

// --- Attachment point list -------------------------------------------------

// An AttachmentPointList consists of a count of the attachment points on a single
// glyph (PointCount) and an array of contour indices of those points (PointIndex),
// listed in increasing numerical order.
type AttachmentPointList struct {
	Coverage           GlyphRange
	Count              int
	attachPointOffsets fontBinSegm
}
