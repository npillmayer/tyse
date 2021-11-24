package otquery

import (
	"testing"

	"github.com/npillmayer/schuko/schukonf/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"github.com/npillmayer/tyse/core/locate/resources"
	"golang.org/x/text/language"
)

func TestScriptMatch(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	f := loadTestFont(t, "calibri")
	otf, err := ot.Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	tracer().Infof("========= loading done =================")
	scr, err := language.ParseScript("Latn")
	if err != nil {
		t.Fatal(err)
	}
	SupportsScript(otf, scr)
	t.Fail()
}

// ---------------------------------------------------------------------------

func loadTestFont(t *testing.T, pattern string) *ot.Font {
	level := tracer().GetTraceLevel()
	tracer().SetTraceLevel(tracing.LevelInfo)
	defer tracer().SetTraceLevel(level)
	//
	otf := &ot.Font{}
	if pattern == "fallback" {
		otf.F = font.FallbackFont()
	} else {
		conf := testconfig.Conf{
			"fontconfig": "/usr/local/bin/fc-list",
			"app-key":    "tyse-test",
		}
		loader := resources.ResolveTypeCase(conf, pattern, font.StyleNormal, font.WeightNormal, 10.0)
		tyc, err := loader.TypeCase()
		if err != nil {
			t.Fatal(err)
		}
		otf.F = tyc.ScalableFontParent()
	}
	t.Logf("loaded font = %s", otf.F.Fontname)
	return otf
}
