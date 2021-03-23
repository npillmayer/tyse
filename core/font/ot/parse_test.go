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
	otf, err := Parse(f.F.Binary)
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
	otf, err := Parse(f.F.Binary)
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
	otf, err := Parse(f.F.Binary)
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
	t.Logf("otf.GPOS: %d features:", gpos.Features.Count())
	for i, ft := range gpos.Features.Tags() {
		t.Logf("[%d] feature '%s'", i, ft)
	}
	if gpos.Features.Count() != 27 {
		t.Errorf("expected 41 features, have %d", gpos.Features.Count())
	}
	t.Logf("otf.GPOS: %d scripts:", gpos.Scripts.Count())
	_ = gpos.Scripts.Tags()
	// t.Logf("otf.GPOS: %d scripts:", len(gpos.scripts))
	// for i, sc := range gpos.scripts {
	// 	t.Logf("[%d] script '%s'", i, sc.Tag)
	// }
	// if len(gpos.scripts) != 3 ||
	// 	gpos.scripts[len(gpos.scripts)-1].Tag.String() != "latn" {
	// 	t.Errorf("expected scripts[2] to be 'latn', isn't")
	// }
	t.Fail()
}

func TestParseGSub(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadTestFont(t, "gentiumplus")
	otf, err := Parse(f.F.Binary)
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
	t.Logf("otf.GSUB: %d features:", gsub.Features.Count())
	for i, ft := range gsub.Features.Tags() {
		t.Logf("[%d] feature '%s'", i, ft)
	}
	if gsub.Features.Count() != 41 {
		t.Errorf("expected 41 features, have %d", gsub.Features.Count())
	}
	// t.Logf("otf.GSUB: %d scripts:", len(gsub.scripts))
	// for i, sc := range gsub.scripts {
	// 	t.Logf("[%d] script '%s'", i, sc.Tag)
	// }
	// if len(gsub.scripts) != 4 ||
	// 	gsub.scripts[len(gsub.scripts)-1].Tag.String() != "latn" {
	// 	t.Errorf("expected scripts[4] to be 'latn', isn't")
	// }
}

func TestParseKern(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadTestFont(t, "calibri")
	otf, err := Parse(f.F.Binary)
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
	otf, err := Parse(f.F.Binary)
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
