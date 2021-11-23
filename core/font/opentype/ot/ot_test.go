package ot

import (
	"testing"

	"github.com/npillmayer/schuko/schukonf/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core/font"
	"github.com/npillmayer/tyse/core/locate/resources"
)

func TestLookupRecordTypeString(t *testing.T) {
	if GSubLookupTypeChainingContext.GSubString() != "Chaining" {
		t.Errorf("expected GSUB_LUTYPE_Reverse_chaining to have string 'Chaining', has %s",
			GSubLookupTypeChainingContext.GSubString())
	}
	if GSubLookupTypeReverseChaining.GSubString() != "Reverse" {
		t.Errorf("expected GSUB_LUTYPE_Reverse_chaining to have string 'Reverse', has %s",
			GSubLookupTypeReverseChaining.GSubString())
	}
	if GPosLookupTypeMarkToLigature.GPosString() != "MarkToLigature" {
		t.Errorf("expected GPOS_LUTYPE_MarkToLigature to have string 'MarkToLigature', has %s",
			GPosLookupTypeMarkToLigature.GPosString())
	}
}

func TestTags(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
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
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	tb := tableBase{}
	tb.name = 0x636d6170
	s := tb.Self().NameTag().String()
	if s != "cmap" {
		t.Errorf("expected table name to be cmap, is %v", s)
	}
}

// ---------------------------------------------------------------------------

func loadTestFont(t *testing.T, pattern string) *Font {
	level := tracer().GetTraceLevel()
	tracer().SetTraceLevel(tracing.LevelInfo)
	defer tracer().SetTraceLevel(level)
	//
	//var err error
	otf := &Font{}
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
	// fontreader := bytes.NewReader(otf.f.Binary)
	// otf.ot, err = sfnt.StrictParse(fontreader)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	t.Logf("loaded font = %s", otf.F.Fontname)
	return otf
}
