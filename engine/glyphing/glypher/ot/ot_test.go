package ot

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ConradIrwin/font/sfnt"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/core/locate/resources"
)

func TestOTFParse(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	otf := loadFallbackFont(t)
	if !strings.Contains(otf.ot.String(), "head") {
		t.Errorf("expected loaded font to have table 'head', doesn't")
	}
}

// ---------------------------------------------------------------------------

func loadFallbackFont(t *testing.T) *OTFont {
	level := trace().GetTraceLevel()
	trace().SetTraceLevel(tracing.LevelInfo)
	defer trace().SetTraceLevel(level)
	//
	var err error
	otf := &OTFont{}
	//otf.f = font.FallbackFont()
	loader := resources.ResolveTypeCase("gentiumplus", font.StyleNormal, font.WeightNormal, 10.0)
	tyc, err := loader.TypeCase()
	if err != nil {
		t.Fatal(err)
	}
	otf.f = tyc.ScalableFontParent()
	fontreader := bytes.NewReader(otf.f.Binary)
	otf.ot, err = sfnt.StrictParse(fontreader)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("loaded font = %s", otf.f.Fontname)
	return otf
}
