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
// OpenType specifies two such tables–GPOS and GSUB–which share some of their
// structure.
type LayoutTable struct {
	ScriptList Navigator
	//FeatureList NavList
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
func (h *LayoutHeader) offsetFor(which layoutTableSectionName) int {
	switch which {
	case layoutScriptSection:
		return int(h.offsets.ScriptListOffset)
	case layoutFeatureSection:
		return int(h.offsets.FeatureListOffset)
	case layoutLookupSection:
		return int(h.offsets.LookupListOffset)
	case layoutFeatureVariationsSection:
		return int(h.offsets.FeatureVariationsOffset)
	}
	tracer().Errorf("illegal section offset type into layout table: %d", which)
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

// layoutTableSectionName lists the sections of OT layout tables, i.e. GPOS and GSUB.
type layoutTableSectionName int

const (
	layoutScriptSection layoutTableSectionName = iota
	layoutFeatureSection
	layoutLookupSection
	layoutFeatureVariationsSection
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

func newGDefTable(tag Tag, b binarySegm, offset, size uint32) *GDefTable {
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
	tracer().Errorf("illegal section offset type into GDEF table: %d", which)
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

func newBaseTable(tag Tag, b binarySegm, offset, size uint32) *BaseTable {
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
type Coverage struct {
	coverageHeader
	GlyphRange GlyphRange
}

type coverageHeader struct {
	CoverageFormat uint16
	Count          uint16
}

func buildGlyphRangeFromCoverage(chead coverageHeader, b binarySegm) GlyphRange {
	tracer().Debugf("coverage format = %d, count = %d", chead.CoverageFormat, chead.Count)
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
	clz := cdf.valueArray.Get(int(glyph - cdf.start)).U16(0)
	return int(clz)
}

type classDefinitionsFormat2 struct {
	count       int   // number of records
	classRanges array // array of ClassRangeRecords — ordered by startGlyphID
}

func (cdf *classDefinitionsFormat2) Lookup(glyph GlyphIndex) int {
	//trace().Debugf("lookup up glyph %d in class def format 2", glyph)
	for i := 0; i < cdf.count; i++ {
		rec := cdf.classRanges.Get(i)
		if glyph < GlyphIndex(rec.U16(0)) {
			return 0
		}
		if glyph < GlyphIndex(rec.U16(2)) {
			return int(rec.U16(4))
		}
	}
	return 0
}

func (cdef *ClassDefinitions) makeArray(b binarySegm, numEntries int, format uint16) array {
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
		tracer().Errorf("illegal format %d of class definition table", format)
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
	// if r[0] == 0xffff {
	// 	r[0] = 0
	// }
	for i := 0; i < lsys.featureIndices.length; i++ {
		if i < 0 || (i+1)*lsys.featureIndices.recordSize > len(lsys.featureIndices.loc.Bytes()) {
			i = 0
		}
		b, _ := lsys.featureIndices.loc.view(i*lsys.featureIndices.recordSize, lsys.featureIndices.recordSize)
		//trace().Debugf("r.i+1[%d] = %d", i+1, u16(b))
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
	attachPointOffsets binarySegm
}

// --- Feature ---------------------------------------------------------------

// Features define the functionality of an OpenType Layout font and they are named to convey
// meaning to the text-processing client. Consider a feature named 'liga' to create ligatures.
// Because of its name, the client knows what the feature does and can decide whether to
// apply it. Font developers can use these features, as well as create their own.
type feature struct {
	err     error
	params  NavLink
	lookups array
}

// Link links to the feature's parameters buffer.
func (f feature) Link() NavLink {
	return f.params
}

func (f feature) Map() NavMap {
	return tagRecordMap16{}
}

func (f feature) List() NavList {
	r := make([]uint16, f.lookups.length)
	for i := 0; i < f.lookups.length; i++ {
		if i < 0 || (i+1)*f.lookups.recordSize > len(f.lookups.loc.Bytes()) {
			i = 0
		}
		b, _ := f.lookups.loc.view(i*f.lookups.recordSize, f.lookups.recordSize)
		r[i] = u16(b)
	}
	return u16List(r)
}

func (f feature) IsVoid() bool {
	return f.lookups.length == 0
}

func (f feature) Error() error {
	return f.err
}

func (f feature) Name() string {
	return "Feature"
}

var _ Navigator = feature{}

// --- Lookup tables ---------------------------------------------------------

// A LookupList table contains an array of offsets to Lookup tables (lookupOffsets).
// The font developer defines the Lookup sequence in the Lookup array to control the order
// in which a text-processing client applies lookup data to glyph substitution or
// positioning operations. (See
// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#lookup-list-table).
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
// LookupList implements the NavList interface.
type LookupList struct {
	array
	base         binarySegm
	lookupsCache []Lookup
	name         string
	err          error
}

func (ll LookupList) Name() string {
	return ll.name
}

/*
func (ll LookupList) Get(i int) NavLocation {
	if i < 0 || i >= ll.length {
		return fontBinSegm{}
	}
	return ll.UnsafeGet(i)
}
*/

// Navigate will navigate to Lookup i in the list.
func (ll LookupList) Navigate(i int) Lookup {
	// acts like NavLink
	if ll.err != nil {
		return Lookup{}
	}
	if ll.lookupsCache == nil {
		ll.lookupsCache = make([]Lookup, ll.length)
	} else if ll.lookupsCache[i].Type != 0 { // type 0 is illegal, i.e. uninitialized
		return ll.lookupsCache[i]
	}
	lookupPtr := ll.Get(i)
	lookup := ll.base[lookupPtr.U16(0):]
	ll.lookupsCache[i] = viewLookup(lookup)
	tracer().Debugf("cached new lookup #%d of type %d", i, ll.lookupsCache[i].Type)
	return ll.lookupsCache[i]
}

var _ NavList = LookupList{}

func GSubLookupType(ltype LayoutTableLookupType) LayoutTableLookupType {
	return ltype & 0x00ff
}

func GPosLookupType(ltype LayoutTableLookupType) LayoutTableLookupType {
	return (ltype & 0xff00) >> 8
}

func MaskGPosLookupType(ltype LayoutTableLookupType) LayoutTableLookupType {
	return ltype << 8
}

func IsGPosLookupType(ltype LayoutTableLookupType) bool {
	return ltype&0xff00 > 0
}

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
// Lookup implements the NavMap interface.
type Lookup struct {
	lookupInfo
	err              error
	loc              NavLocation      // offset start for sub-tables
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
	tracer().Debugf("lookup location has size %d", b.Size())
	if b.Size() < 10 {
		return Lookup{}
	}
	lookup := Lookup{loc: b}
	lookup.Type = LayoutTableLookupType(b.U16(0))
	lookup.Flag = LayoutTableLookupFlag(b.U16(2))
	lookup.SubTableCount = b.U16(4)
	// r := b.Reader()
	// if err := binary.Read(r, binary.BigEndian, &lookup.lookupInfo); err != nil {
	// 	tracer().Errorf("corrupt Lookup table")
	// 	return Lookup{} // nothing sensible to to except to return empty table
	// }
	tracer().Debugf("Lookup has %d sub-tables", lookup.SubTableCount)
	//
	var err error
	lookup.subTables, err = parseArray16(b.Bytes(), 4, "Lookup", "Lookup-Subtables")
	if err != nil {
		tracer().Errorf("corrupt Lookup table")
		return Lookup{} // nothing sensible to to except to return empty table
	}
	if b.Size() >= 4+lookup.subTables.Size()+2 {
		lookup.markFilteringSet = b.U16(4 + lookup.subTables.Size())
	}
	//trace().Debugf("lookup has type %s", lookup.Type.GSubString())
	return lookup
}

func (l Lookup) Subtable(i int) *LookupSubtable {
	if l.err != nil || i >= int(l.SubTableCount) {
		return nil
	}
	if l.subTablesCache == nil {
		l.subTablesCache = make([]LookupSubtable, l.SubTableCount)
		for i := 0; i < l.subTables.length; i++ {
			n := l.subTables.Get(i).U16(0) // offset to subtable[i]
			tracer().Debugf("lookup subtable at offset %d", n)
			link := makeLink16(n, l.loc.Bytes(), "LookupSubtable") // wrap offset into link
			loc := link.Jump()
			b := binarySegm(loc.Bytes())
			l.subTablesCache[i] = parseLookupSubtable(b, l.Type)
		}
	}
	return &l.subTablesCache[i]
}

// Lookup returns a byte segment as output of applying lookup l to input glyph g.
// g is shortened from 32-bit to 16-bit by using the low bits.
//
// If g is not identified as applicable for the lookup feature, an emtpy byte segment
// is returned.
func (l Lookup) Lookup(g uint32) NavLocation {
	// inx, ok := l.coverage.GlyphRange.Lookup(GlyphIndex(g >> 16))
	// if !ok {
	// 	return fontBinSegm{}
	// }
	// trace().Debugf("lookup of 0x%x -> %d", g, inx)
	return binarySegm{} // TODO
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

// LookupSubtable is a type for OpenType Lookup Subtables, which are the basis for Lookup operations
// (see https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#lookup-table).
//
// “Each LookupType may occur in one or more subtable formats. The ‘best’ format depends on
// the type of substitution and the resulting storage efficiency. When glyph information
// is best presented in more than one format, a single lookup may define more than
// one subtable, as long as all the subtables are for the same LookupType.”
//
// The interpretation of the Index-elements and the Support field depend heavily on the
// type of the lookup-subtable. For example, for a GSUB lookup of type 'Single Substitution Format 1'
// Support will be interpreted as a delta and be added to glyph IDs, while lookup type
// 'Ligature Substitution Format 1' will ignore the Support field and repeatedly descend into
// the Index tables to match glyph sequences suitable for ligature substitution. (see
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#lookuptype-1-single-substitution-subtable).
// Package `ot` will not do this interpretation, but rather leave it to higher-protocol packages.
type LookupSubtable struct {
	LookupType LayoutTableLookupType // may differ from Lookup.Type for Type=Extension
	Format     uint16                // lookup subtables may come in more than one format
	Coverage   Coverage              // for which glyphs is this lookup applicable
	Index      VarArray              // Index tables/arrays to lookup up substitutions/positions
	Support    interface{}           // some lookup variants use additional data
}

// LookupType 5: Contextual Substitution Subtable
//
// A Contextual Substitution subtable describes glyph substitutions in context that replace one
// or more glyphs within a certain pattern of glyphs.
// Contextual substitution subtables can use any of three formats that are common to the GSUB
// and GPOS tables. These define input sequence patterns to be matched against the text glyphs
// sequence, and then actions to be applied to glyphs within the input sequence. The actions
// are specified as “nested” lookups, and each is applied to a particular sequence positions
// within the input sequence.
// Each sequence position + nested lookup combination is specified in a SequenceLookupRecord.
/*

=> otlayout/feature.go

func gsubLookupType5Fmt1(l *Lookup, lksub *LookupSubtable, g GlyphIndex) NavLocation {
	inx, ok := lksub.Coverage.GlyphRange.Match(g)
	if !ok {
		return fontBinSegm{}
	}
	return lookupAndReturn(lksub.Index, inx, true) // returns a LigatureSet
}
*/

// lookupAndReturn is a small helper which looks up an index for a glyph (previously
// returned from a coverage table), checks for errors, and returns the resulting bytes.
// TODO check that this is inlined by the compiler.
func lookupAndReturn(index VarArray, ginx int, deep bool) NavLocation {
	outglyph, err := index.Get(ginx, deep)
	if err != nil {
		return binarySegm{}
	}
	return outglyph
}

// SequenceContext is a type for identifying the input sequence context for
// contextual lookups (see
// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#common-structures-for-contextual-lookup-subtables).
//
// Clients will receive this struct in the `Support` field of a LookupSubtable, whenever it's appropriate:
//
// ▪︎ GSUB Lookup Type 5 and 6, format 2 and 3
//
// ▪︎ GPOS Lookup Type 5 and 7, format 2 and 3
//
// For type 5, the length of each non-void slice will be exactly 1; for type 6/7 they may be of
// arbitrary length. Its exact allocation will depend on the type/format combination of the lookup
// subtable.
type SequenceContext struct {
	BacktrackCoverage []Coverage         // for format 3
	InputCoverage     []Coverage         // for format 3
	LookaheadCoverage []Coverage         // for format 3
	ClassDefs         []ClassDefinitions // for format 2
}

// sequenceContext is a type for identifying the input sequence context for
// contextual lookups.
// https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#sequence-context-format-1-simple-glyph-contexts
type xsequenceContext struct {
	format        uint16           // 1, 2 or 3
	coverage      []Coverage       // for all formats
	classDef      ClassDefinitions // for format 2
	rules         varArray         // for format 1 and 2
	lookupRecords array            // for format 3
}

// The glyphCount value is the total number of glyphs in the input sequence, including the
// first glyph. The inputSequence array specifies the remaining glyphs in the input sequence,
// in order. (The glyph at inputSequence index 0 corresponds to glyph sequence index 1.)
//
// The seqLookupRecords array lists the sequence lookup records that specify actions to be
// taken on glyphs at various positions within the input sequence. These do not have to be
// ordered in sequence position order; they are ordered according to the desired result.
// All of the sequence lookup records are processed in order, and each applies to the
// results of the actions indicated by the preceding record.
type sequenceRule struct {
	glyphCount    uint16
	inputSequence array
	lookupRecords array
}
