package ot

import (
	"encoding/binary"
)

/*
From https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2:

OpenType Layout consists of five tables: the Glyph Substitution table (GSUB),
the Glyph Positioning table (GPOS), the Baseline table (BASE),
the Justification table (JSTF), and the Glyph Definition table (GDEF).
These tables use some of the same data formats.
*/

// --- Layout tables ---------------------------------------------------------

// LayoutTable is a base type for layout tables.
// OpenType specifies two such tables–GPOS and GSUB–which share some of their
// structure.
type LayoutTable struct {
	ScriptList  TagRecordMap
	FeatureList TagRecordMap
	LookupList  LookupList
	header      *LayoutHeader
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

// offsetFor returns an offset for a layout table section within the layout table
// (GPOS or GSUB).
// A layout table contains four sections:
// ▪︎ Script Section,
// ▪︎ Feature Section,
// ▪︎ Lookup Section,
// ▪︎ Feature Variations Section.
// (see type LayoutTableSectionName)
//
func (h *LayoutHeader) offsetFor(which LayoutTableSectionName) int {
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

// --- Layout tables sections ------------------------------------------------

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

// --- GDEF table ------------------------------------------------------------

// GDefTable, the Glyph Definition (GDEF) table, provides various glyph properties
// used in OpenType Layout processing.
//
// See also
// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#class-definition-table
type GDefTable struct {
	tableBase
	header                 GDefHeader
	GlyphClassDef          ClassDefinitions
	AttachmentPointList    AttachmentPointList
	MarkAttachmentClassDef ClassDefinitions
	MarkGlyphSets          []GlyphRange
}

func newGDefTable(tag Tag, b fontBinSegm, offset, size uint32) *GDefTable {
	t := &GDefTable{}
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
//type GDefTableSectionName int

// GDefGlyphClassDefSection GDefTableSectionName = iota
// GDefAttachListSection
// GDefLigCaretListSection
// GDefMarkAttachClassSection
// GDefMarkGlyphSetsDefSection
// GDefItemVarStoreSection

// Sections of a GDEF table.
const (
	GDefGlyphClassDefSection    = "GlyphClassDef"
	GDefAttachListSection       = "AttachList"
	GDefLigCaretListSection     = "LigCaretList"
	GDefMarkAttachClassSection  = "MarkAttachClassDef"
	GDefMarkGlyphSetsDefSection = "MarkGlyphSetsDef"
	GDefItemVarStoreSection     = "ItemVarStore"
)

// offsetFor returns an offset for a table section within the GDEF table.
// A GDEF table contains six sections:
// ▪︎ glyph class definitions,
// ▪︎ attachment list definitions,
// ▪︎ ligature carets lists,
// ▪︎ mark attachment class definitions,
// ▪︎ mark glyph sets definitions,
// ▪︎ item variant section.
// (see https://docs.microsoft.com/en-us/typography/opentype/spec/gdef#gdef-header)
//
func (h GDefHeader) offsetFor(which string) int {
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
	tableBase
	axisTables [2]AxisTable
}

func newBaseTable(tag Tag, b fontBinSegm, offset, size uint32) *BaseTable {
	t := &BaseTable{}
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

type AxisTable struct {
	baselineTags      tagList
	baseScriptRecords TagRecordMap
}

// --- Coverage table module -------------------------------------------------

// Covarage denotes an indexed set of glyphs.
// Each LookupSubtable (except an Extension LookupType subtable) in a lookup references
// a Coverage table (Coverage), which specifies all the glyphs affected by a
// substitution or positioning operation described in the subtable.
// The GSUB, GPOS, and GDEF tables rely on this notion of coverage. If a glyph does
// not appear in a Coverage table, the client can skip that subtable and move
// immediately to the next subtable.
//
type Coverage struct {
	coverageHeader
	GlyphRange GlyphRange
}

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

// GlyphClassDefEnum lists the glyph classes for ClassDefinitions
// ('GlyphClassDef'-table).
type GlyphClassDefEnum uint16

const (
	BaseGlyph      GlyphClassDefEnum = iota //single character, spacing glyph
	LigatureGlyph                           //multiple character, spacing glyph
	MarkGlyph                               //non-spacing combining glyph
	ComponentGlyph                          //part of single character, spacing glyph
)

// ClassDefinitions groups glyphs into classes, denoted as integer values.
//
// From the spec:
// For efficiency and ease of representation, a font developer can group glyph indices
// to form glyph classes. Class assignments vary in meaning from one lookup subtable
// to another. For example, in the GSUB and GPOS tables, classes are used to describe
// glyph contexts. GDEF tables also use the idea of glyph classes.
// (see https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#class-definition-table)
type ClassDefinitions struct {
	format  uint16          // format version 1 or 2
	records classDefVariant // either format 1 or 2
}

func (cdef *ClassDefinitions) setRecords(recs array, startGlyphID GlyphIndex) {
	if cdef.format == 1 {
		cdef.records = &classDefinitionsFormat1{
			count:      recs.length,
			start:      startGlyphID,
			valueArray: recs,
		}
	} else if cdef.format == 2 {
		cdef.records = &classDefinitionsFormat2{
			count:       recs.length,
			classRanges: recs,
		}
	}
}

type classDefVariant interface {
	Lookup(GlyphIndex) int
}

type classDefinitionsFormat1 struct {
	count      int        // number of entries
	start      GlyphIndex // glyph ID of the first entry in a format-1 table
	valueArray array      // array of Class Values — one per glyph ID
}

func (cdf *classDefinitionsFormat1) Lookup(glyph GlyphIndex) int {
	if glyph < cdf.start || glyph >= cdf.start+GlyphIndex(cdf.count) {
		return 0
	}
	clz := cdf.valueArray.UnsafeGet(int(glyph - cdf.start)).U16(0)
	return int(clz)
}

type classDefinitionsFormat2 struct {
	count       int   // number of records
	classRanges array // array of ClassRangeRecords — ordered by startGlyphID
}

func (cdf *classDefinitionsFormat2) Lookup(glyph GlyphIndex) int {
	for i := 0; i < cdf.count; i++ {
		rec := cdf.classRanges.UnsafeGet(i)
		if glyph < GlyphIndex(rec.U16(0)) {
			return 0
		}
		if glyph < GlyphIndex(rec.U16(2)) {
			return int(rec.U16(4))
		}
	}
	return 0
}

func (cdef *ClassDefinitions) makeArray(b fontBinSegm, numEntries int, format uint16) array {
	var size, recsize int
	switch format {
	case 1:
		recsize = 2
		size = 6 + numEntries*recsize
		b = b[6:size]
	case 2:
		recsize = 6
		size = 4 + numEntries*recsize
		b = b[4:size]
	default:
		trace().Errorf("illegal format %d of class definition table", format)
		return array{}
	}
	return array{recordSize: recsize, length: numEntries, loc: b}
}

// Lookup returns the class defined for a glyph, or 0 (= default class).
func (cdef *ClassDefinitions) Lookup(glyph GlyphIndex) int {
	return cdef.records.Lookup(glyph)
}

// --- LangSys table ---------------------------------------------------------

type langSys struct {
	err            error
	mandatory      uint16 // 0xffff if unused
	featureIndices array  // list of uint16 indices
}

func (lsys langSys) Link() NavLink {
	return nullLink("LangSys records not linkable")
}

func (lsys langSys) Map() NavMap {
	return tagRecordMap16{}
}

// entry 0 will be the mandatory feature
func (lsys langSys) List() NavList {
	r := make([]uint16, lsys.featureIndices.length+1)
	r[0] = lsys.mandatory
	for i := 0; i < lsys.featureIndices.length; i++ {
		if i < 0 || (i+1)*lsys.featureIndices.recordSize > len(lsys.featureIndices.loc.Bytes()) {
			i = 0
		}
		b, _ := lsys.featureIndices.loc.view(i*lsys.featureIndices.recordSize, lsys.featureIndices.recordSize)
		r[i+1] = u16(b)
	}
	return u16List(r)
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

// --- Lookup tables ---------------------------------------------------------

// A LookupList table contains an array of offsets to Lookup tables (lookupOffsets).
// The font developer defines the Lookup sequence in the Lookup array to control the order
// in which a text-processing client applies lookup data to glyph substitution or
// positioning operations.
//
// Lookup tables are essential for implementing the various OpenType font features.
// The details are sometimes tricky and it's often hard to remember how a lookup type
// works, if you're not doing it on a daily basis. As this is a low-level package,
// we focus on decoding the sub-tables for lookups and on abstracting the details
// of the specific variants of GSUB- and GPOS-lookup away, offering a map-like behaviour.
//
// Lookups depend on sub-tables to do the actual work, which in turn may occur in
// various format versions. This package implements all table/sub-table variants defined by
// the OT spec, together with the algorithms to access their functionality.
//
// Other packages working on top of `ot` should abstract this further and operate
// in terms of OT features, hiding altogether the existence of lookup lists and lookups from
// clients.
//
type LookupList struct {
	array
	err error
}

func (ll LookupList) Len() int {
	return ll.length
}

func (ll LookupList) Get(i int) NavLocation {
	if i < 0 || i >= ll.length {
		return fontBinSegm{}
	}
	return ll.UnsafeGet(i)
}

// Navigate will navigate to Lookup i in the list.
func (ll LookupList) Navigate(i int) Lookup {
	// acts like NavLink
	if ll.err != nil {
		return Lookup{}
	}
	lookup := ll.Get(i)
	return viewLookup(lookup)
}

var _ NavList = LookupList{}

// Lookup tables are contained in a LookupList.
// A Lookup table defines the specific conditions, type, and results of a substitution or
// positioning action that is used to implement a feature. For example, a substitution
// operation requires a list of target glyph indices to be replaced, a list of replacement
// glyph indices, and a description of the type of substitution action.
//
// Each Lookup table may contain only one type of information (LookupType), determined by
// whether the lookup is part of a GSUB or GPOS table. GSUB supports eight LookupTypes,
// and GPOS supports nine LookupTypes
//
type Lookup struct {
	lookupInfo
	err              error
	subTables        array            // Array of offsets to lookup subrecords, from beginning of Lookup table
	markFilteringSet uint16           // Index (base 0) into GDEF mark glyph sets structure. This field is only present if bit useMarkFilteringSet of lookup flags is set.
	subTablesCache   []LookupSubtable // cache for sub-tables already parsed and called
}

// header information for Lookup table
type lookupInfo struct {
	Type          LayoutTableLookupType
	Flag          LayoutTableLookupFlag
	SubTableCount uint16 // Number of subtables for this lookup
}

// viewLookup reads a Lookup from the bytes of a NavLocation. It first parses the
// lookupInfo and after that parses the subtable record list.
func viewLookup(b NavLocation) Lookup {
	if b.Size() < 10 {
		return Lookup{}
	}
	r := b.Reader()
	lookup := Lookup{}
	if err := binary.Read(r, binary.BigEndian, &lookup.lookupInfo); err != nil {
		trace().Errorf("corrupt Lookup table")
		return Lookup{} // nothing sensible to to except to return empty table
	}
	trace().Debugf("Lookup has %d sub-tables", lookup.SubTableCount)
	//
	var err error
	lookup.subTables, err = parseArray16(b.Bytes(), 4)
	if err != nil {
		trace().Errorf("corrupt Lookup table")
		return Lookup{} // nothing sensible to to except to return empty table
	}
	if b.Size() >= 4+lookup.subTables.Size()+2 {
		lookup.markFilteringSet = b.U16(4 + lookup.subTables.Size())
	}
	return lookup
}

// Lookup returns a byte segment as output of applying lookup l to input glyph g.
// g is shortened from 32-bit to 16-bit by using the low bits.
//
// If g is not identified as applicable for the lookup feature, an emtpy byte segment
// is returned.
//
func (l Lookup) Lookup(g uint32) NavLocation {
	// inx, ok := l.coverage.GlyphRange.Lookup(GlyphIndex(g >> 16))
	// if !ok {
	// 	return fontBinSegm{}
	// }
	// trace().Debugf("lookup of 0x%x -> %d", g, inx)
	return fontBinSegm{} // TODO
}

func (l Lookup) Name() string {
	return "Lookup"
}

// LookupTag is not defined for Lookup and will return a void link.
func (l Lookup) LookupTag(tag Tag) NavLink {
	return nullLink("cannot lookup tag in Lookup")
}

// IsTagRecordMap returns false
func (l Lookup) IsTagRecordMap() bool {
	return false
}

// AsTagRecordMap returns an empty TagRecordMap
func (l Lookup) AsTagRecordMap() TagRecordMap {
	return tagRecordMap16{}
}

var _ NavMap = Lookup{}

// Each LookupType may occur in one or more subtable formats. The “best” format depends on
// the type of substitution and the resulting storage efficiency. When glyph information
// is best presented in more than one format, a single lookup may define more than
// one subtable, as long as all the subtables are for the same LookupType.
type LookupSubtable struct {
	format   uint16
	coverage Coverage
	index    varArray
	support  interface{} // TODO make this a more specific interface
}

// GSUB LookupType 1: Single Substitution Subtable
//
// Single substitution (SingleSubst) subtables tell a client to replace a single glyph
// with another glyph. The subtables can be either of two formats. Both formats require
// two distinct sets of glyph indices: one that defines input glyphs (specified in the
// Coverage table), and one that defines the output glyphs.

// GSUB LookupSubtable Type 1 Format 1 calculates the indices of the output glyphs, which
// are not explicitly defined in the subtable. To calculate an output glyph index,
// Format 1 adds a constant delta value to the input glyph index. For the substitutions to
// occur properly, the glyph indices in the input and output ranges must be in the same order.
// This format does not use the Coverage index that is returned from the Coverage table.
//
func gsubLookupType1Fmt1(l *Lookup, lksub *LookupSubtable, g GlyphIndex) NavLocation {
	_, ok := lksub.coverage.GlyphRange.Lookup(g)
	if !ok {
		return fontBinSegm{}
	}
	// support is deltaGlyphID: add to original glyph ID to get substitute glyph ID
	delta := lksub.support.(GlyphIndex)
	return uintBytes(uint16(g + delta))
}

// GSUB LookupSubtable Type 1 Format 2 provides an array of output glyph indices
// (substituteGlyphIDs) explicitly matched to the input glyph indices specified in the
// Coverage table.
//
// The substituteGlyphIDs array must contain the same number of glyph indices as the
// Coverage table. To locate the corresponding output glyph index in the substituteGlyphIDs
// array, this format uses the Coverage index returned from the Coverage table.
//
func gsubLookupType1Fmt2(l *Lookup, lksub *LookupSubtable, g GlyphIndex) NavLocation {
	inx, ok := lksub.coverage.GlyphRange.Lookup(g)
	if !ok {
		return fontBinSegm{}
	}
	return lookupAndReturn(&lksub.index, inx, true)
}

// LookupType 2: Multiple Substitution Subtable
//
// A Multiple Substitution (MultipleSubst) subtable replaces a single glyph with more
// than one glyph, as when multiple glyphs replace a single ligature.

// GSUB LookupSubtable Type 2 Format 1 defines a count of offsets in the sequenceOffsets
// array (sequenceCount), and an array of offsets to Sequence tables that define the output
// glyph indices (sequenceOffsets). The Sequence table offsets are ordered by the Coverage
// index of the input glyphs.
// For each input glyph listed in the Coverage table, a Sequence table defines the output
// glyphs. Each Sequence table contains a count of the glyphs in the output glyph sequence
// (glyphCount) and an array of output glyph indices (substituteGlyphIDs).
func gsubLookupType2Fmt1(l *Lookup, lksub *LookupSubtable, g GlyphIndex) NavLocation {
	inx, ok := lksub.coverage.GlyphRange.Lookup(g)
	if !ok {
		return fontBinSegm{}
	}
	return lookupAndReturn(&lksub.index, inx, true)
}

// LookupType 3: Alternate Substitution Subtable
//
// An Alternate Substitution (AlternateSubst) subtable identifies any number of aesthetic
// alternatives from which a user can choose a glyph variant to replace the input glyph.
// For example, if a font contains four variants of the ampersand symbol, the 'cmap' table
// will specify the index of one of the four glyphs as the default glyph index, and an
// AlternateSubst subtable will list the indices of the other three glyphs as alternatives.
// A text-processing client would then have the option of replacing the default glyph with
// any of the three alternatives.

// GSUB LookupSubtable Type 3 Format 1: For each glyph, an AlternateSet subtable contains a
// count of the alternative glyphs (glyphCount) and an array of their glyph indices
// (alternateGlyphIDs).
func gsubLookupType3Fmt1(l *Lookup, lksub *LookupSubtable, g GlyphIndex) NavLocation {
	inx, ok := lksub.coverage.GlyphRange.Lookup(g)
	if !ok {
		return fontBinSegm{}
	}
	return lookupAndReturn(&lksub.index, inx, true)
}

// LookupType 4: Ligature Substitution Subtable
//
// A Ligature Substitution (LigatureSubst) subtable identifies ligature substitutions where
// a single glyph replaces multiple glyphs. One LigatureSubst subtable can specify any number
// of ligature substitutions.

// GSUB LookupSubtable Type 4 Format 1 receives a sequence of glyphs and outputs a
// single glyph replacing the sequence. The Coverage table specifies only the index of the
// first glyph component of each ligature set.
//
// As this is a multi-lookup algorithm, calling gsubLookupType4Fmt1 will return a
// NavLocation which is a LigatureSet, i.e. a list of records of unequal lengths.
//
func gsubLookupType4Fmt1(l *Lookup, lksub *LookupSubtable, g GlyphIndex) NavLocation {
	inx, ok := lksub.coverage.GlyphRange.Lookup(g)
	if !ok {
		return fontBinSegm{}
	}
	return lookupAndReturn(&lksub.index, inx, true) // returns a LigatureSet
}

// LigatureSetLookup trys to match a sequence of glyph IDs to the pattern portions
// ('components') of every Ligature of a LigatureSet, and if a match is found,
// returns the ligature glyph for the pattern.
//
// loc is a byte segment usually returned from a call to a type 4 (Ligature Substitution)
// GSUB lookup.
//
// The resulting glyph should replace a sequence of glyphs from the input code-points
// including the initial glyph input to the type 4 Ligature Substitution, continued
// with the glyphs provided to LigatureSetLookup.
//
func LigatureSetLookup(loc NavLocation, glyphs []GlyphIndex) GlyphIndex {
	// loc shoud be at a LigatureSet
	ligset, err := parseArray16(loc.Bytes(), 0)
	if err != nil {
		return 0
	}
	// iterate over all Ligature entries, pointed to by an offset16
	for i := 0; i < ligset.length; i++ {
		ptr := ligset.UnsafeGet(i).U16(0)
		// Ligature table (glyph components for one ligature):
		// uint16  ligatureGlyph     glyph ID of ligature to substitute
		// uint16  componentCount    Number of components in the ligature
		// uint16  componentGlyphIDs[componentCount-1]    Array of component glyph IDs
		ligglyph := GlyphIndex(loc.U16(int(ptr)))
		compCount := loc.U16(int(ptr) + 2)
		comps := array{
			recordSize: 6, // 3 * sizeof(uint16)
			length:     int(compCount) - 1,
			loc:        loc.Bytes()[ptr:],
		}
		if len(glyphs) != comps.length {
			break
		}
		match := true
		for i, g := range glyphs {
			if g != GlyphIndex(comps.UnsafeGet(i).U16(0)) {
				match = false
				break
			}
		}
		if match {
			return ligglyph
		}
	}
	return 0
}

// lookupAndReturn is a small helper which looks up an index for a glyph (previously
// returned from a coverage table), checks for errors, and returns the resulting bytes.
// TODO check that this is inlined by the compiler.
func lookupAndReturn(index *varArray, ginx int, deep bool) NavLocation {
	outglyph, err := index.Get(ginx, deep)
	if err != nil {
		return fontBinSegm{}
	}
	return outglyph
}
