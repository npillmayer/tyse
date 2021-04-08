package otlyt

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/core/font/ot"
	"github.com/npillmayer/tyse/core/locate/resources"
)

func TestTagRegistry(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	typ, err := identifyFeatureTag(ot.T("liga"))
	if err != nil {
		t.Fatal(err)
	}
	if typ != GSubFeatureType {
		t.Errorf("expected 'liga' to be found as a GSUB feature, isn't")
	}
}

func TestCalibriCMap(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	otf := parseFont(t, "calibri")
	t.Logf("Using font %s for test", otf.F.Fontname)
	cmap := otf.Table(ot.T("cmap")).Self().AsCMap()
	r, pos := rune('('), ot.GlyphIndex(4)
	glyphID := cmap.GlyphIndexMap.Lookup(r)
	t.Logf("found glyph ID %#x for %#U", glyphID, r)
	if glyphID != pos {
		t.Errorf("expected %#U to be on glyph position %d, is %d", r, pos, glyphID)
	}
	t.Fail()
}

func TestFeatureList(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	otf := parseFont(t, "calibri")
	t.Logf("Using font %s for test", otf.F.Fontname)
	// Calibri has no DFLT feature set
	gsubFeats, gposFeats, err := FontFeatures(otf, ot.T("latn"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(gsubFeats) == 0 {
		t.Errorf("GSUB features not found in font Calibri")
	}
	if len(gposFeats) == 0 {
		t.Errorf("GPOS features not found in font Calibri")
	}
	t.Logf("found %d GSUB features", len(gsubFeats))
	t.Logf("found %d GPOS features", len(gposFeats))
	if len(gsubFeats) != 24 {
		t.Errorf("expected Calibri to have 24 GSUB features for 'latn', has %d", len(gsubFeats))
	}
	featcase := gsubFeats[1]
	if featcase == nil || featcase.Tag() != ot.T("case") {
		t.Errorf("expected feature #1 to be 'case', isn't")
	}
	t.Logf("# of lookups for 'case' = %d", featcase.LookupCount())
	t.Logf("index of lookup #0 for 'case' = %d", featcase.LookupIndex(0))
	if featcase.LookupIndex(0) != 9 {
		t.Errorf("expected index of lookup #0 of feature 'case' to be 9, isn't")
	}
}

func TestFeatureCase(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	otf := parseFont(t, "calibri")
	t.Logf("Using font %s for test", otf.F.Fontname)
	// Calibri has no DFLT feature set
	gsubFeats, _, err := FontFeatures(otf, ot.T("latn"), 0)
	if err != nil || len(gsubFeats) < 2 {
		t.Errorf("GSUB feature 'case' not found in font Calibri")
	}
	featcase := gsubFeats[1]
	if featcase == nil || featcase.Tag() != ot.T("case") {
		t.Errorf("expected feature #1 to be 'case', isn't")
	}
	t.Logf("# of lookups for 'case' = %d", featcase.LookupCount())
	t.Logf("index of lookup #0 for 'case' = %d", featcase.LookupIndex(0))
	if featcase.LookupIndex(0) != 9 {
		t.Errorf("expected index of lookup #0 of feature 'case' to be 9, isn't")
	}
	_, applied, buf := ApplyFeature(otf, featcase, prepareGlyphBuffer("@", otf, t), 0)
	t.Logf("Application of 'case' returned glyph buffer %v", buf)
	if !applied {
		t.Error("feature 'case' not applied")
	}
	if buf[0] != 925 {
		t.Errorf("expected 'case' to replace '@' with glyph 925, have %d", buf[0])
	}
}

// ---------------------------------------------------------------------------

func prepareGlyphBuffer(s string, otf *ot.Font, t *testing.T) []ot.GlyphIndex {
	cmap := otf.Table(ot.T("cmap")).Self().AsCMap()
	runes := []rune(s)
	buf := make([]ot.GlyphIndex, len(runes))
	for i, r := range runes {
		glyphID := cmap.GlyphIndexMap.Lookup(r)
		buf[i] = glyphID
	}
	return buf
}

func parseFont(t *testing.T, pattern string) *ot.Font {
	sfnt := loadTestFont(t, pattern)
	if sfnt == nil {
		return nil
	}
	otf, err := ot.Parse(sfnt.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	otf.F = sfnt.F
	t.Logf("--- font parsed ---")
	return otf
}

func loadTestFont(t *testing.T, pattern string) *ot.Font {
	level := trace().GetTraceLevel()
	trace().SetTraceLevel(tracing.LevelInfo)
	defer trace().SetTraceLevel(level)
	otf := &ot.Font{}
	if pattern == "fallback" {
		otf.F = font.FallbackFont()
	} else {
		loader := resources.ResolveTypeCase(pattern, font.StyleNormal, font.WeightNormal, 10.0)
		tyc, err := loader.TypeCase()
		if err != nil {
			t.Fatal(err)
		}
		otf.F = tyc.ScalableFontParent()
	}
	t.Logf("loaded font = %s", otf.F.Fontname)
	return otf
}
