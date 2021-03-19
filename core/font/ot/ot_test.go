package ot

import (
	"bytes"
	"testing"

	"github.com/ConradIrwin/font/sfnt"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/core/locate/resources"
)

func TestLookupRecordTypeString(t *testing.T) {
	if GSUB_LUTYPE_Chaining_Context.GSubString() != "Chaining" {
		t.Errorf("expected GSUB_LUTYPE_Reverse_chaining to have string 'Chaining', has %s",
			GSUB_LUTYPE_Chaining_Context.GSubString())
	}
	if GSUB_LUTYPE_Reverse_Chaining.GSubString() != "Reverse" {
		t.Errorf("expected GSUB_LUTYPE_Reverse_chaining to have string 'Reverse', has %s",
			GSUB_LUTYPE_Reverse_Chaining.GSubString())
	}
	if GPOS_LUTYPE_MarkToLigature.GPosString() != "MarkToLigature" {
		t.Errorf("expected GPOS_LUTYPE_MarkToLigature to have string 'MarkToLigature', has %s",
			GPOS_LUTYPE_MarkToLigature.GPosString())
	}
}

func TestTags(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	tag := Tag(0x636d6170)
	if tag.String() != "cmap" {
		t.Errorf("expected tag 0x636d6170 to be 'cmap', is %s", tag.String())
	}
	tag = MakeTag([]byte("cmap"))
	if tag.String() != "cmap" {
		t.Errorf("expected tag MakeTag(cmap) to be 'cmap', is %s", tag.String())
	}
	tag = T("cmap")
	if tag.String() != "cmap" {
		t.Errorf("expected tag T(cmap) to be 'cmap', is %s", tag.String())
	}
}

func TestTableName(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	tb := TableBase{}
	tb.name = 0x636d6170
	s := tb.String()
	if s != "cmap" {
		t.Errorf("expected table name to be cmap, is %v", s)
	}
}

// ---------------------------------------------------------------------------

func loadTestFont(t *testing.T, pattern string) *Font {
	level := trace().GetTraceLevel()
	trace().SetTraceLevel(tracing.LevelInfo)
	defer trace().SetTraceLevel(level)
	//
	var err error
	otf := &Font{}
	if pattern == "fallback" {
		otf.f = font.FallbackFont()
	} else {
		loader := resources.ResolveTypeCase(pattern, font.StyleNormal, font.WeightNormal, 10.0)
		tyc, err := loader.TypeCase()
		if err != nil {
			t.Fatal(err)
		}
		otf.f = tyc.ScalableFontParent()
	}
	fontreader := bytes.NewReader(otf.f.Binary)
	otf.ot, err = sfnt.StrictParse(fontreader)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("loaded font = %s", otf.f.Fontname)
	return otf
}
