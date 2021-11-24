package otquery

import (
	"fmt"
	"testing"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
	"golang.org/x/text/language"
)

func TestFontTypeInfo(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	tracing.Select("tyse.fonts").SetTraceLevel(tracing.LevelInfo)
	//
	otf := loadCalibri(t)
	fti := FontType(otf)
	if fti != "TrueType" {
		t.Errorf("expected font type string to be 'TrueType', is %q", fti)
	}
}

func TestGeneralInfo(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	tracing.Select("tyse.fonts").SetTraceLevel(tracing.LevelInfo)
	//
	otf := loadCalibri(t)
	info := NameInfo(otf, language.AmericanEnglish)
	t.Logf("info = %v", info)
	if fam, ok := info["family"]; !ok {
		t.Fatal("cannot find family name in font Calibri")
	} else if fam != "Calibri" {
		t.Fatalf("expected font family to equal Calibri, is %q", fam)
	}
}

func TestLayoutInfo(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	tracing.Select("tyse.fonts").SetTraceLevel(tracing.LevelInfo)
	//
	otf := loadCalibri(t)
	lt := LayoutTables(otf)
	t.Logf("layout tables: %v", lt)
	if fmt.Sprintf("%v", lt) != "[GDEF GSUB GPOS]" {
		t.Errorf("expected Calibri to have [GDEF GSUB GPOS], is %v", lt)
	}
}

// ----------------------------------------------------------------------

func loadCalibri(t *testing.T) *ot.Font {
	f := loadTestFont(t, "calibri")
	otf, err := ot.Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	return otf
}
