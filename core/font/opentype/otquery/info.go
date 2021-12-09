package otquery

import (
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"golang.org/x/text/language"
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
//
func NameInfo(otf *ot.Font, lang language.Tag) map[string]string {
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

// GlyphClass collects glyph class information for a glyph index.
type GlyphClass struct {
	Class           int
	MarkAttachClass int
	MarkGlyphSet    int
}

// GlyphClasses retrieves glyph class information for a given glyph index.
func GlyphClasses(otf *ot.Font, gid ot.GlyphIndex) GlyphClass {
	t := otf.Table(ot.T("GDEF"))
	if t == nil {
		return GlyphClass{}
	}
	gdef := t.Self().AsGDef()
	clz := GlyphClass{
		Class:           gdef.GlyphClassDef.Lookup(gid),
		MarkAttachClass: gdef.MarkAttachmentClassDef.Lookup(gid),
	}
	for _, set := range gdef.MarkGlyphSets {
		if n, ok := set.Match(gid); ok {
			clz.MarkGlyphSet = n
		}
	}
	return clz
}
