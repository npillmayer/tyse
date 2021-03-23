package otquery

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/core/font/ot"
	"github.com/npillmayer/tyse/core/locate/resources"
	"golang.org/x/text/language"
)

func TestScriptMatch(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f := loadTestFont(t, "calibri")
	otf, err := ot.Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	trace().Infof("========= loading done =================")
	scr, err := language.ParseScript("Latn")
	if err != nil {
		t.Fatal(err)
	}
	SupportsScript(otf, scr)
	t.Fail()
}

// ---------------------------------------------------------------------------

func loadTestFont(t *testing.T, pattern string) *ot.Font {
	level := trace().GetTraceLevel()
	trace().SetTraceLevel(tracing.LevelInfo)
	defer trace().SetTraceLevel(level)
	//
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
