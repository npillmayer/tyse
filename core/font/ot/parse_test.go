package ot

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
)

func TestTableName(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	tb := TableBase{}
	tb.name = 0x636d6170
	s := tb.String()
	if s != "cmap" {
		t.Errorf("expected table name to be cmap, is %v", s)
	}
}

func TestParseHeader(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadFallbackFont(t)
	otf, err := Parse(f.f.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("otf.header.tag = %x", otf.header.FontType)
	t.Fail()
}

func TestParseFont(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadFallbackFont(t)
	otf, err := Parse(f.f.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("otf.header.tag = %x", otf.header.FontType)
	t.Fail()
}

// ---------------------------------------------------------------------------

func GposDebugInfo(otf *OTFont) {
	level := trace().GetTraceLevel()
	trace().SetTraceLevel(tracing.LevelInfo)
	defer trace().SetTraceLevel(level)
	_, err := otf.ot.GposTable()
	if err != nil {
		trace().Errorf("cannot read GPOS table of OpenType font %s", otf.f.Fontname)
		trace().Errorf(err.Error())
		return
	}
	trace().Infof("OpenType GPOS table of %s", otf.f.Fontname)
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
