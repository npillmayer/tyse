package ot

import (
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core"
)

func TestParseHeader(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	f := loadTestFont(t, "gentiumplus")
	otf, err := Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("otf.header.tag = %x", otf.Header.FontType)
	if otf.Header.FontType != 0x00010000 {
		t.Fatalf("expected font Gentium to be OT 0x0001000, is %x", otf.Header.FontType)
	}
}

// TODO TODO
func TestCMapTableGlyphIndex(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	otf := parseFont(t, "calibri")
	t.Logf("otf.header.tag = %x", otf.Header.FontType)
	table := getTable(otf, "cmap", t)
	cmap := table.Self().AsCMap()
	if cmap == nil {
		t.Fatal("cannot convert cmap table")
	}
	r := rune('A')
	glyph := cmap.GlyphIndexMap.Lookup(r)
	if glyph == 0 {
		t.Error("expected glyph position for 'A', got 0")
	}
	t.Logf("glyph ID = %d | 0x%x", glyph, glyph)
	if glyph != 4 {
		t.Errorf("expected glyph position for 'A' to be 4, got %d", glyph)
	}
}

func TestParseGPos(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	f := loadTestFont(t, "calibri")
	otf, err := Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("font contains tables:")
	hasGPos := false
	for _, tag := range otf.TableTags() {
		t.Logf("  %s", tag.String())
		if tag.String() == "GPOS" {
			hasGPos = true
		}
	}
	if !hasGPos {
		t.Fatalf("expected font to have GPOS table, hasn't")
	}
	gposTag := T("GPOS")
	gpos := otf.tables[gposTag].Self().AsGPos()
	if gpos == nil {
		t.Fatalf("cannot find a GPOS table")
	}
	t.Logf("otf.GPOS: %d features:", gpos.FeatureList.Len())
	if gpos.FeatureList.Len() != 27 {
		t.Errorf("expected 27 GPOS features, have %d", gpos.FeatureList.Len())
	}
	t.Logf("otf.GPOS: %d scripts:", gpos.ScriptList.Map().AsTagRecordMap().Len())
	_ = gpos.ScriptList.Map().AsTagRecordMap().Tags()
	// t.Logf("otf.GPOS: %d scripts:", len(gpos.scripts))
	// for i, sc := range gpos.scripts {
	// 	t.Logf("[%d] script '%s'", i, sc.Tag)
	// }
	// if len(gpos.scripts) != 3 ||
	// 	gpos.scripts[len(gpos.scripts)-1].Tag.String() != "latn" {
	// 	t.Errorf("expected scripts[2] to be 'latn', isn't")
	// }
}

func TestParseGSub(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	f := loadTestFont(t, "gentiumplus")
	otf, err := Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("font contains tables:")
	hasGSub := false
	for _, tag := range otf.TableTags() {
		t.Logf("  %s", tag.String())
		if tag.String() == "GSUB" {
			hasGSub = true
		}
	}
	if !hasGSub {
		t.Fatalf("expected font to have GSUB table, hasn't")
	}
	gsubTag := T("GSUB")
	gsub := otf.tables[gsubTag].Self().AsGSub()
	if gsub == nil {
		t.Fatalf("cannot find a GSUB table")
	}
	t.Logf("otf.GSUB: %d features:", gsub.FeatureList.Len())
	if gsub.FeatureList.Len() != 41 {
		t.Errorf("expected 41 features, have %d", gsub.FeatureList.Len())
	}
	// t.Logf("otf.GSUB: %d scripts:", len(gsub.scripts))
	// for i, sc := range gsub.scripts {
	// 	t.Logf("[%d] script '%s'", i, sc.Tag)
	// }
	// if len(gsub.scripts) != 4 ||
	// 	gsub.scripts[len(gsub.scripts)-1].Tag.String() != "latn" {
	// 	t.Errorf("expected scripts[4] to be 'latn', isn't")
	// }
}

func TestParseKern(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	f := loadTestFont(t, "calibri")
	otf, err := Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("font contains tables:")
	hasKern := false
	for _, tag := range otf.TableTags() {
		t.Logf("  %s", tag.String())
		if tag.String() == "kern" {
			hasKern = true
		}
	}
	if !hasKern {
		t.Fatalf("expected font to have kern table, hasn't")
	}
	kern := otf.tables[T("kern")].Self().AsKern()
	if kern == nil {
		t.Fatalf("cannot find a kern table")
	}
}

func TestParseOtherTables(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	f := loadTestFont(t, "calibri")
	otf, err := Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	maxp := otf.tables[T("maxp")].Self().AsMaxP()
	if maxp == nil {
		t.Fatalf("cannot find a maxp table")
	}
	t.Logf("MaxP.NumGlyphs = %d", maxp.NumGlyphs)
	if maxp.NumGlyphs != 3874 {
		t.Errorf("expected Calibri to have 3874 glyphs, but %d indicated", maxp.NumGlyphs)
	}
	loca := otf.tables[T("loca")].Self().AsLoca()
	if loca == nil {
		t.Fatalf("cannot find a maxp table")
	}
	hhea := otf.tables[T("hhea")].Self().AsHHea()
	if hhea == nil {
		t.Fatalf("cannot find a hhea table")
	}
	t.Logf("hhea number of metrics = %d", hhea.NumberOfHMetrics)
	if hhea.NumberOfHMetrics != 3843 {
		t.Errorf("expected Calibri to have 3843 metrics, but %d indicated", hhea.NumberOfHMetrics)
	}
}

func TestParseGDef(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	otf := parseFont(t, "calibri")
	table := getTable(otf, "GDEF", t)
	gdef := table.Self().AsGDef()
	if gdef.GlyphClassDef.format == 0 {
		t.Fatalf("GDEF table has not GlyphClassDef section")
	}
	// Calibri uses glyph class def format 2
	t.Logf("GDEF.GlyphClassDef.Format = %d", gdef.GlyphClassDef.format)
	glyph := GlyphIndex(1380) // ID of uni0336 in Calibri
	clz := gdef.GlyphClassDef.Lookup(glyph)
	t.Logf("gylph class for uni0336|1280 is %d", clz)
	if clz != 3 {
		t.Errorf("expected to be uni0336 of class 3 (mark), is %d", clz)
	}
}

func TestParseGSubLookups(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	otf := parseFont(t, "calibri")
	table := getTable(otf, "GSUB", t)
	gsub := table.Self().AsGSub()
	ll := gsub.LookupList
	if ll.err != nil {
		t.Fatal(ll.err)
	} else if ll.array.length == 0 {
		t.Fatalf("GSUB table has no LookupList section")
	}
	/*
	   <LookupList>
	     <!-- LookupCount=49 -->      Calibri has 49 lookup entries
	     <Lookup index="0">           Lookup #0 is of type 7, extending to type 1
	       <LookupType value="7"/>
	       <LookupFlag value="0"/>
	       <!-- SubTableCount=1 -->
	       <ExtensionSubst index="0" Format="1">
	         <ExtensionLookupType value="1"/>
	         <SingleSubst Format="2">
	           <Substitution in="Scedilla" out="uni0218"/>
	           <Substitution in="scedilla" out="uni0219"/>
	           <Substitution in="uni0162" out="uni021A"/>
	           <Substitution in="uni0163" out="uni021B"/>
	         </SingleSubst>
	       </ExtensionSubst>
	     </Lookup>
	*/
	t.Logf("font Calibri has %d lookups", ll.array.length)
	lookup := gsub.LookupList.Navigate(0)
	t.Logf("lookup[0].subTables count is %d", lookup.subTables.length)
	if lookup.subTablesCache == nil {
		t.Logf("no cached sub-tables")
	}
	st := lookup.Subtable(0)
	t.Logf("type of sub-table is %s", st.LookupType.GSubString())
	if st.LookupType != 1 {
		t.Errorf("expected first lookup to be of type 7 -> 1, is %d", st.LookupType)
	}
}

// ---------------------------------------------------------------------------

func getTable(otf *Font, name string, t *testing.T) Table {
	table := otf.tables[T(name)]
	if table == nil {
		t.Fatalf("table %s not found in font", name)
	}
	return table
}

func parseFont(t *testing.T, pattern string) *Font {
	otf := loadTestFont(t, pattern)
	if otf == nil {
		return nil
	}
	otf, err := Parse(otf.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	t.Logf("--- font parsed ---")
	return otf
}
