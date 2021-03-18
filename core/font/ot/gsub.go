package ot

import (
	"strconv"

	"github.com/npillmayer/schuko/tracing"
)

// GSubTable is a type representing an OpenType GSUB table
// (see https://docs.microsoft.com/en-us/typography/opentype/spec/gsub).
type GSubTable struct {
	TableBase
	header  *LayoutHeader
	lookups []*lookupRecord
}

// Base returns the enclosed TableBase type this table inherits from.
func (g *GSubTable) Base() *TableBase {
	return &g.TableBase
}

var _ Table = &GSubTable{}

// GSUB Table Lookup Type
// https://docs.microsoft.com/en-us/typography/opentype/spec/gsub#table-organization

// GSUB LookupType Enumeration
const (
	GSUB_LUTYPE_Single           uint16 = 1 // Replace one glyph with one glyph
	GSUB_LUTYPE_Multiple         uint16 = 2 // Replace one glyph with more than one glyph
	GSUB_LUTYPE_Alternate        uint16 = 3 // Replace one glyph with one of many glyphs
	GSUB_LUTYPE_Ligature         uint16 = 4 // Replace multiple glyphs with one glyph
	GSUB_LUTYPE_Context          uint16 = 5 // Replace one or more glyphs in context
	GSUB_LUTYPE_Chaining_Context uint16 = 6 // Replace one or more glyphs in chained context
	GSUB_LUTYPE_Extension_Subs   uint16 = 7 // Extension mechanism for other substitutions
	GSUB_LUTYPE_Reverse_chaining uint16 = 8 // Applied in reverse order, replace single glyph in chaining context
)

func GSubLookupTypeString(lutype uint16) string {
	switch lutype {
	case GSUB_LUTYPE_Single:
		return "GSUB_Single"
	case GSUB_LUTYPE_Multiple:
		return "GSUB_Multiple"
	case GSUB_LUTYPE_Ligature:
		return "GSUB_Ligature"
	case GSUB_LUTYPE_Alternate:
		return "GSUB_Alternate"
	case GSUB_LUTYPE_Context:
		return "GSUB_Context"
	case GSUB_LUTYPE_Chaining_Context:
		return "GSUB_ChainingContext"
	}
	return strconv.Itoa(int(lutype))
}

func GSubDebugInfo(otf *OTFont) {
	level := trace().GetTraceLevel()
	trace().SetTraceLevel(tracing.LevelInfo)
	defer trace().SetTraceLevel(level)
	gsub, err := otf.ot.GsubTable()
	if err != nil {
		trace().Errorf("cannot read GSUB table of OpenType font %s", otf.f.Fontname)
		trace().Errorf(err.Error())
		return
	}
	trace().Infof("OpenType GSUB table of %s", otf.f.Fontname)
	trace().Infof("scripts:")
	for _, script := range gsub.Scripts {
		trace().Infof("   script = %s", script.String())
	}
	trace().Infof("features:")
	for _, feature := range gsub.Features {
		trace().Infof("   feature = %s", feature.String())
	}
	trace().Infof("lookups:")
	for _, lookup := range gsub.Lookups {
		trace().Infof("   lookup = %s", GSubLookupTypeString(lookup.Type))
	}
}
