package ot

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
)

func TestParseHeader(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadTestFont(t, "gentiumplus")
	otf, err := Parse(f.f.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("otf.header.tag = %x", otf.Header.FontType)
	if otf.Header.FontType != 0x00010000 {
		t.Fatalf("expected font Gentium to be OT 0x0001000, is %x", otf.Header.FontType)
	}
}

// TODO TODO
func TestCMapTableGlyphIndex(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadTestFont(t, "gentiumplus")
	otf, err := Parse(f.f.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("otf.header.tag = %x", otf.Header.FontType)
	t.Error("TODO CMap Test check")
}

func TestParseGPos(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadTestFont(t, "calibri")
	otf, err := Parse(f.f.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("font contains tables:")
	hasGPos := false
	for _, tag := range otf.TableTags() {
		t.Logf("  %s", tag.String())
		if tag.String() == "GPOS" {
			hasGPos = true
		}
	}
	if !hasGPos {
		t.Fatalf("expected font to have GPOS table, hasn't")
	}
	gposTag := T("GPOS")
	gpos := otf.tables[gposTag].Base().AsGPos()
	if gpos == nil {
		t.Fatalf("cannot find a GPOS table")
	}
	t.Logf("otf.GPOS: %d features:", len(gpos.features))
	for i, ft := range gpos.features {
		t.Logf("[%d] feature '%s'", i, ft.Tag)
	}
	if len(gpos.features) != 27 ||
		gpos.features[len(gpos.features)-1].Tag.String() != "mkmk" {
		t.Errorf("expected features[26] to be 'mkmk', isn't")
	}
	t.Logf("otf.GPOS: %d scripts:", len(gpos.scripts))
	for i, sc := range gpos.scripts {
		t.Logf("[%d] script '%s'", i, sc.Tag)
	}
	if len(gpos.scripts) != 3 ||
		gpos.scripts[len(gpos.scripts)-1].Tag.String() != "latn" {
		t.Errorf("expected scripts[2] to be 'latn', isn't")
	}
}

func TestParseGSub(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadTestFont(t, "gentiumplus")
	otf, err := Parse(f.f.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("font contains tables:")
	hasGSub := false
	for _, tag := range otf.TableTags() {
		t.Logf("  %s", tag.String())
		if tag.String() == "GSUB" {
			hasGSub = true
		}
	}
	if !hasGSub {
		t.Fatalf("expected font to have GSUB table, hasn't")
	}
	gsubTag := T("GSUB")
	gsub := otf.tables[gsubTag].Base().AsGSub()
	if gsub == nil {
		t.Fatalf("cannot find a GSUB table")
	}
	t.Logf("otf.GSUB: %d features:", len(gsub.features))
	for i, ft := range gsub.features {
		t.Logf("[%d] feature '%s'", i, ft.Tag)
	}
	if len(gsub.features) != 41 ||
		gsub.features[len(gsub.features)-1].Tag.String() != "ss07" {
		t.Errorf("expected features[40] to be 'ss07', isn't")
	}
	t.Logf("otf.GSUB: %d scripts:", len(gsub.scripts))
	for i, sc := range gsub.scripts {
		t.Logf("[%d] script '%s'", i, sc.Tag)
	}
	if len(gsub.scripts) != 4 ||
		gsub.scripts[len(gsub.scripts)-1].Tag.String() != "latn" {
		t.Errorf("expected scripts[4] to be 'latn', isn't")
	}
}

func TestParseKern(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadTestFont(t, "calibri")
	otf, err := Parse(f.f.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("font contains tables:")
	hasKern := false
	for _, tag := range otf.TableTags() {
		t.Logf("  %s", tag.String())
		if tag.String() == "kern" {
			hasKern = true
		}
	}
	if !hasKern {
		t.Fatalf("expected font to have kern table, hasn't")
	}
	kern := otf.tables[T("kern")].Base().AsKern()
	if kern == nil {
		t.Fatalf("cannot find a kern table")
	}
}

func TestParseOtherTables(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadTestFont(t, "calibri")
	otf, err := Parse(f.f.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	maxp := otf.tables[T("maxp")].Base().AsMaxP()
	if maxp == nil {
		t.Fatalf("cannot find a maxp table")
	}
	t.Logf("MaxP.NumGlyphs = %d", maxp.NumGlyphs)
	if maxp.NumGlyphs != 3874 {
		t.Errorf("expected Calibri to have 3874 glyphs, but %d indicated", maxp.NumGlyphs)
	}
	loca := otf.tables[T("loca")].Base().AsLoca()
	if loca == nil {
		t.Fatalf("cannot find a maxp table")
	}
	hhea := otf.tables[T("hhea")].Base().AsHHea()
	if hhea == nil {
		t.Fatalf("cannot find a hhea table")
	}
	t.Logf("hhea number of metrics = %d", hhea.NumberOfHMetrics)
	if hhea.NumberOfHMetrics != 3843 {
		t.Errorf("expected Calibri to have 3843 metrics, but %d indicated", hhea.NumberOfHMetrics)
	}
}

// ---------------------------------------------------------------------------

func GposDebugInfo(otf *Font) {
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
