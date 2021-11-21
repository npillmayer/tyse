package font

import (
	"fmt"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	xfont "golang.org/x/image/font"
)

type sw struct {
	s xfont.Style
	w xfont.Weight
}

func TestGuess(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	for k, v := range map[string]sw{
		"fonts/Clarendon-bold.ttf":               {xfont.StyleNormal, xfont.WeightBold},
		"Microsoft/Gill Sans MT Bold Italic.ttf": {xfont.StyleItalic, xfont.WeightBold},
		"Cambria Math.ttf":                       {xfont.StyleNormal, xfont.WeightNormal},
	} {
		style, weight := GuessStyleAndWeight(k)
		t.Logf("style = %d, weight = %d", style, weight)
		if style != v.s || weight != v.w {
			t.Errorf("expected different style or weight for %s", k)
		}
	}
}

func TestMatch(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	if !Matches("fonts/Clarendon-bold.ttf",
		"clarendon", xfont.StyleNormal, xfont.WeightBold) {
		t.Errorf("expected match for Clarendon, haven't")
	}
	if !Matches("Microsoft/Gill Sans MT Bold Italic.ttf",
		"gill sans", xfont.StyleItalic, xfont.WeightBold) {
		t.Errorf("expected match for Gill, haven't")
	}
	if !Matches("Cambria Math.ttf",
		"cambria", xfont.StyleNormal, xfont.WeightNormal) {
		t.Errorf("expected match for Cambria Math, haven't")
	}
}

func TestNormalizeFont(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "resources")
	defer teardown()
	//
	n := NormalizeFontname("Clarendon", xfont.StyleItalic, xfont.WeightBold)
	if n != "clarendon-italic-bold" {
		t.Errorf("expected different normalized name for clarendon")
	}
}

func TestOpenOpenTypeCaseCreation(t *testing.T) {
	//fontpath := locate.FileResource("GentiumPlus-R.ttf", "font")
	fontpath := "../locate/resources/packaged/fonts/GentiumPlus-R.ttf"
	f, err := LoadOpenTypeFont(fontpath)
	if err != nil {
		t.Fatal(err)
	}
	tc, err := f.PrepareCase(12.0)
	if err != nil {
		t.Logf("cannot create OT face for [%s]\n", f.Fontname)
		t.Fatal(err)
	}
	metrics := tc.font.Metrics()
	fmt.Printf("interline spacing for [%s]@%.1fpt is %s\n", f.Fontname, tc.size, metrics.Height)
}
