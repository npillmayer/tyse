package resources

import (
	"strings"
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
	font.GlobalRegistry().LogFontList()
}

var fclist = `
/System/Library/Fonts/Supplemental/NotoSansGothic-Regular.ttf: Noto Sans Gothic:style=Regular
/System/Library/Fonts/NotoSerifMyanmar.ttc: Noto Serif Myanmar,Noto Serif Myanmar Light:style=Light,Regular
/System/Library/Fonts/Supplemental/NotoSansCarian-Regular.ttf: Noto Sans Carian:style=Regular
/System/Library/Fonts/NotoSansMyanmar.ttc: Noto Sans Zawgyi:style=Regular
/System/Library/Fonts/Supplemental/NotoSansSylotiNagri-Regular.ttf: Noto Sans Syloti Nagri:style=Regular
/System/Library/Fonts/NotoNastaliq.ttc: Noto Nastaliq Urdu:style=Bold
/System/Library/Fonts/Supplemental/NotoSansCham-Regular.ttf: Noto Sans Cham:style=Regular
/System/Library/Fonts/NotoSansArmenian.ttc: Noto Sans Armenian:style=Bold
`

func TestFCBinary(t *testing.T) {
	teardown := testconfig.QuickConfig(t, map[string]string{
		"fontconfig": "/usr/local/bin/fc-list",
	})
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	fc, err := findFontConfigBinary()
	t.Logf("fontconfig binary = %s", fc)
	if err != nil {
		t.Error(err)
	}
	if !strings.HasSuffix(fc, "fc-list") {
		t.Errorf("fontconfig parameter does not point to fc-list")
	}
}

func TestFCList(t *testing.T) {
	teardown := testconfig.QuickConfig(t, map[string]string{
		"fontconfig": "/usr/local/bin/fc-list",
		"app-key":    "tyse-test",
	})
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	l, ok := cacheFontConfigList(false)
	if !ok {
		t.Errorf("cannot cache fontconfig list")
	}
	t.Logf("l = %s", l)
}

func TestFCLoad(t *testing.T) {
	teardown := testconfig.QuickConfig(t, map[string]string{
		"fontconfig": "/usr/local/bin/fc-list",
		"app-key":    "tyse-test",
	})
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	list, ok := loadFontConfigList()
	if !ok {
		t.Fatalf("cannot load fontconfig list")
	}
	t.Logf("found %d fonts in fontconfig list", len(list))
	f, v := findFontConfigFont("new york", xfont.StyleItalic, xfont.WeightNormal)
	t.Logf("found font = %s in variant %s", f.Family, v)
	if f.Family != "New York" {
		t.Errorf("expected to find font New York, found %v", f)
	}
}

func TestFCFind(t *testing.T) {
	teardown := testconfig.QuickConfig(t, map[string]string{
		"fontconfig": "/usr/local/bin/fc-list",
		"app-key":    "tyse-test",
	})
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	f, v := findFontConfigFont("new york", xfont.StyleItalic, xfont.WeightNormal)
	t.Logf("found font = (%s) in variant (%s)", f.Family, v)
	t.Logf("font file = %s", f.Path)
	if f.Family != "New York" {
		t.Errorf("expected to find font New York, found %v", f)
	}
}
