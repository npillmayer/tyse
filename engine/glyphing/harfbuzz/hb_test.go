package harfbuzz_test

import (
	"fmt"
	"strings"
	"testing"

	hb "github.com/benoitkugler/textlayout/harfbuzz"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/engine/glyphing"
	"github.com/npillmayer/tyse/engine/glyphing/harfbuzz"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/text/language"
)

func TestHBScript(t *testing.T) {
	id := "Plrd"
	script := language.MustParseScript(id)
	hb_script := harfbuzz.Script4HB(script)
	hstr := fmt.Sprintf("%x", uint32(hb_script))
	if hstr != "706c7264" {
		t.Logf("script %q: %x => %x", id, script, uint32(hb_script))
		t.Errorf("expected HB script of 706c7264, is %s", hstr)
	}
}

func TestHBLang(t *testing.T) {
	l := "de_DE"
	langT, err := language.Parse(l)
	if err != nil {
		t.Error(err)
	}
	h := harfbuzz.Lang4HB(langT)
	if h != "de-de" {
		t.Logf("Go lang = %v", langT)
		t.Logf("HB lang = %v, expected de-de", h)
		t.Fail()
	}
}

func TestHBDir(t *testing.T) {
	var d glyphing.Direction = glyphing.TopToBottom
	dir := harfbuzz.Direction4HB(d)
	if dir != hb.TopToBottom {
		t.Errorf("expected dir to be %d, is %d", hb.TopToBottom, dir)
	}
}

func TestHBShape(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.glyphs")
	defer teardown()
	//
	input := "Hello"
	text := strings.NewReader(input)
	font := loadGoFont(t)
	params := glyphing.Params{
		Font: font,
	}
	seq, err := harfbuzz.Shape(text, nil, nil, params)
	if err != nil {
		t.Error(err)
	}
	if seq.Glyphs == nil {
		t.Error("expected shaping output to be non-nil")
	}
	if len(seq.Glyphs) != len(input) {
		t.Errorf("expected %d output glyphs, have %d", len(input), len(seq.Glyphs))
	}
}

// ---------------------------------------------------------------------------

func loadGoFont(t *testing.T) *font.TypeCase {
	gofont := &font.ScalableFont{
		Fontname: "Go Sans",
		Filepath: "internal",
		Binary:   goregular.TTF,
	}
	var err error
	gofont.SFNT, err = sfnt.Parse(gofont.Binary)
	if err != nil {
		t.Fatal("cannot load Go font") // this cannot happen
	}
	typecase, err := gofont.PrepareCase(12.0)
	if err != nil {
		t.Fatal(err)
	}
	return typecase
}
