package ot

import (
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core"
)

func TestNavLink(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	otf := loadCalibri(t)
	table := otf.Table(T("GSUB"))
	if table == nil {
		t.Fatal("cannot locate table GSUB in font")
	}
	gsub := table.Self().AsGSub()
	recname := gsub.ScriptList.Map().LookupTag(T("latn")).Navigate().Name()
	t.Logf("walked to %s", recname)
	lang := gsub.ScriptList.Map().LookupTag(T("latn")).Navigate().Map().AsTagRecordMap().LookupTag(T("TRK"))
	langlist := lang.Navigate().List()
	t.Logf("list is %s of length %v", lang.Name(), langlist.Len())
	if lang.Name() != "LangSys" || langlist.Len() != 24 {
		t.Errorf("expected LangSys[IPPH] to contain 24 feature entries, has %d", langlist.Len())
	}
}

func TestTableNav(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	otf := loadCalibri(t)
	table := otf.Table(T("name"))
	if table == nil {
		t.Fatal("cannot locate table name in font")
	}
	name := table.Fields().Name()
	if name != "name" {
		t.Errorf("expected table to have name 'name', have %s", name)
	}
	key := MakeTag([]byte{3, 1, 0, 1}) // Windows 1-encoded field 1 = Font Family Name
	x := table.Fields().Map().AsTagRecordMap().LookupTag(key).Navigate().Name()
	if x != "Calibri" {
		t.Errorf("expected Windows/1 encoded field 1 to be 'Calibri', is %s", x)
	}
}

func TestTableNavOS2(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.fonts")
	defer teardown()
	//
	otf := loadCalibri(t)
	table := otf.Table(T("OS/2"))
	if table == nil {
		t.Fatal("cannot locate table OS/2 in font")
	}
	name := table.Fields().Name()
	if name != "OS/2" {
		t.Errorf(name)
	}
	loc := table.Fields().List().Get(1)
	if loc.U16(0) != 400 {
		t.Errorf("expected xAvgCharWidth (size %d) of Calibri to be 400, is %d", loc.Size(), loc.U16(0))
	}
}

// ---------------------------------------------------------------------------

func loadCalibri(t *testing.T) *Font {
	f := loadTestFont(t, "calibri")
	otf, err := Parse(f.F.Binary)
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
	tracer().Infof("========= loading done =================")
	return otf
}
