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
}

// ---------------------------------------------------------------------------

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
