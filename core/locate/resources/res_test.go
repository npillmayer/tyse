package resources

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/font"
	xfont "golang.org/x/image/font"
)

func TestLoadImage(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	loader := ResolveImage("placeholder.png", "high")
	img, err := loader.Image()
	if err != nil {
		t.Error(err)
	}
	if img == nil {
		t.Fatalf("img is nil, should be placeholder.png")
	}
	w := img.Bounds().Dx()
	t.Logf("width of image = %d", w)
}

func TestLoadPackagedFont(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	loader := ResolveTypeCase("GentiumPlus", xfont.StyleNormal, xfont.WeightNormal, 11.0)
	//time.Sleep(500)
	typecase, err := loader.TypeCase()
	if err != nil {
		t.Error(err)
	}
	if typecase == nil {
		t.Fatalf("typecase is nil, should not be")
	}
	t.Logf("pt-size of typecase = %f", typecase.PtSize())
	t.Logf("name of typecase = %s", typecase.ScalableFontParent().Fontname)
	if typecase.ScalableFontParent().Fontname != "GentiumPlus-R.ttf" {
		t.Errorf("expected font to be named GentiumPlus-R, isn't")
	}
}

func TestResolveGoogleFont(t *testing.T) {
	teardown := testconfig.QuickConfig(t, map[string]string{
		"app-key": "tyse-test",
	})
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	loader := ResolveTypeCase("Antic", xfont.StyleNormal, xfont.WeightNormal, 11.0)
	typecase, err := loader.TypeCase()
	if err != nil {
		t.Error(err)
	}
	if typecase == nil {
		t.Fatalf("typecase is nil, should not be")
	}
	t.Logf("pt-size of typecase = %f", typecase.PtSize())
	t.Logf("name of typecase = %s", typecase.ScalableFontParent().Fontname)
	font.GlobalRegistry().DebugList()
}
