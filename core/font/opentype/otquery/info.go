package otquery

import (
	"github.com/npillmayer/tyse/core/font/opentype/ot"
)

// FontType returns the font type, encoded in the font header, as a string.
func FontType(otf *ot.Font) string {
	if otf.Header == nil {
		return "<empty>"
	}
	typ := otf.Header.FontType
	switch typ {
	case 0x4f54544f: // OTTO
		return "OpenType (outlines)"
	case 0x00010000: // TrueType
		return "TrueType"
	case 0x74727565: // true
		return "TrueType (Mac legacy)"
	}
	return "<unknown>"
}

// NameInfo returns a map with selected fields from OpenType table `name`.
// Will include (if available in the font) "family", "subfamily", "version".
//
// Parameter `lang` is currently unused.
func NameInfo(otf *ot.Font, lang ot.Tag) map[string]string {
	table := otf.Table(ot.T("name"))
	names := make(map[string]string)
	if table == nil {
		tracer().Debugf("no name table found in font %s", otf.F.Fontname)
		return names
	}
	m := table.Fields().Map().AsTagRecordMap()
	tracer().Debugf("table name = %q", table.Fields().Name())
	for _, tag := range m.Tags() {
		tracer().Debugf("names tag %q", tag.String())
	}
	// font family
	familyKeys := [][]byte{
		{3, 1, 0, 1}, // Windows platform, encoding BMP
		{1, 0, 0, 1}, // Mac platform, encoding Roman
	}
	findKey(table, names, "family", familyKeys)
	// font sub-family
	subFamKeys := [][]byte{
		{3, 1, 0, 2},
		{1, 0, 0, 2},
	}
	findKey(table, names, "subfamily", subFamKeys)
	// font version
	versionKeys := [][]byte{
		{3, 1, 0, 5},
		{1, 0, 0, 5},
	}
	findKey(table, names, "version", versionKeys)
	return names
}

func findKey(table ot.Table, m map[string]string, fieldname string, keys [][]byte) {
	for _, key := range keys {
		key := ot.MakeTag(key)
		val := table.Fields().Map().AsTagRecordMap().LookupTag(key).Navigate().Name()
		if val != "" {
			m[fieldname] = val
			break
		}
	}
}

// LayoutTables returns a list of tag strings, one for each layout-table a font includes.
//
// From the spec:
// OpenType Layout makes use of five tables: GSUB, GPOS, BASE, JSTF, and GDEF.
func LayoutTables(otf *ot.Font) []string {
	var lt []string
	tags := otf.TableTags()
	for _, tag := range tags {
		switch tag.String() {
		case "GSUB", "GPOS", "BASE", "JSTF", "GDEF":
			lt = append(lt, tag.String())
		}
	}
	return lt
}

// GlyphClasses collects glyph class information for a glyph.
//
// From the OpenType spec:
// A Mark Glyph Sets table is used to define sets of mark glyphs that can be used in lookup tables
// within the GSUB or GPOS table to control how mark glyphs within a glyph sequence are treated by
// lookups. Mark glyph sets are used in GSUB and GPOS lookups to filter which marks in a string are
// considered or ignored.
// Mark glyph sets are used for the same purpose as mark attachment classes, which is as filters
// for GSUB and GPOS lookups. Mark glyph sets differ from mark attachment classes, however, in
// that mark glyph sets may intersect as needed by the font developer. As for mark attachment classes,
// only one mark glyph set can be referenced in any given lookup.
type GlyphClasses struct {
	Class          GlyphClass
	MarkAttachment MarkAttachmentClass

	MarkGlyphSet int // fonts may define arbitrary numbers of sets
}

// GlyphClass denotes an OpenType glyph class. From the OpenType Spec:
//
// The Glyph Class Definition (GlyphClassDef) table identifies four types of glyphs in a font: base glyphs,
// ligature glyphs, combining mark glyphs, and glyph components. GSUB and GPOS lookups define and use
// these glyph classes to differentiate the types of glyphs in a string. For example, GPOS uses
// the glyph classes to distinguish between a simple base glyph and the mark glyph that follows it.
// In addition, a client uses class definitions to apply GSUB and GPOS LookupFlag data correctly.
// For example, a LookupFlag may specify ignoring ligatures and marks during a glyph operation.
type GlyphClass int16

const (
	GC_Default        GlyphClass = 0 // unassigned
	GC_BaseGlyph      GlyphClass = 1 // single character, spacing glyph
	GC_LigatureGlyph  GlyphClass = 2 // multiple character, spacing glyph
	GC_MarkGlyph      GlyphClass = 3 // non-spacing combining glyph
	GC_ComponentGlyph GlyphClass = 4 // part of single character, spacing glyph
)

// MarkAttachmentClass denotes an OpenType mark attachment class. From the OpenType spec:
//
// A Mark Attachment Class Definition Table is used to assign mark glyphs into different classes that
// can be used in lookup tables within the GSUB or GPOS table to control how mark glyphs within a glyph
// sequence are treated by lookups.
type MarkAttachmentClass int16

const (
	MAC_Top    MarkAttachmentClass = 1 // top mark
	MAC_Bottom MarkAttachmentClass = 2 // bottom mark
)

// ClassesForGlyph retrieves glyph class information for a given glyph index.
func ClassesForGlyph(otf *ot.Font, gid ot.GlyphIndex) GlyphClasses {
	t := otf.Table(ot.T("GDEF"))
	if t == nil {
		return GlyphClasses{}
	}
	gdef := t.Self().AsGDef()
	clz := GlyphClasses{
		Class:          GlyphClass(gdef.GlyphClassDef.Lookup(gid)),
		MarkAttachment: MarkAttachmentClass(gdef.MarkAttachmentClassDef.Lookup(gid)),
	}
	for _, set := range gdef.MarkGlyphSets {
		if n, ok := set.Match(gid); ok {
			clz.MarkGlyphSet = n
		}
	}
	return clz
}
