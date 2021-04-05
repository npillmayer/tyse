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

func TestLiga(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	otf := parseFont(t, "calibri")
	gsub := getTable(otf, "GSUB", t).Self().AsGSub()
	liga := gsub.FeatureList.LookupTag(ot.T("liga"))
	if liga == nil {
		t.Errorf("liga table not found in font Calibri")
	}
}

// ---------------------------------------------------------------------------

func getTable(otf *ot.Font, name string, t *testing.T) ot.Table {
	table := otf.Table(ot.T(name))
	if table == nil {
		t.Fatalf("table %s not found in font", name)
	}
	return table
}

func parseFont(t *testing.T, pattern string) *ot.Font {
	otf := loadTestFont(t, pattern)
	if otf == nil {
		return nil
	}
	otf, err := ot.Parse(otf.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
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
