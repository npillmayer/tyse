package ot

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
)

func TestLink(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	otf := loadCalibri(t)
	table := otf.Table(T("GSUB"))
	if table == nil {
		t.Fatal("cannot locate table GSUB in font")
	}
	gsub := table.Base().AsGSub()
	recname := gsub.Scripts.Lookup(T("latn")).Navigate().Name()
	t.Logf("walked to %s", recname)
	lang := gsub.Scripts.Lookup(T("latn")).Navigate().Map().Lookup(T("IPPH"))
	langlist := lang.Navigate().List()
	t.Logf("list is %s of length %v", lang.Name(), langlist.Len())
	if lang.Name() != "LangSys" || langlist.Len() != 10 {
		t.Errorf("expected LangSys[IPPH] to contain 10 feature entries, has %d", langlist.Len())
	}
}

func TestTableNav(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	otf := loadCalibri(t)
	table := otf.Table(T("name"))
	if table == nil {
		t.Fatal("cannot locate table name in font")
	}
	name := table.Base().Fields().Name()
	if name != "name" {
		t.Errorf("expected table to have name 'name', have %s", name)
	}
	key := MakeTag([]byte{3, 1, 0, 1}) // Windows 1-encoded field 1 = Font Family Name
	x := table.Base().Fields().Map().Lookup(key).Navigate().Name()
	if x != "Calibri" {
		t.Errorf("expected Windows/1 encoded field 1 to be 'Calibri', is %s", x)
	}
}

func TestTableOS2(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	otf := loadCalibri(t)
	table := otf.Table(T("OS/2"))
	if table == nil {
		t.Fatal("cannot locate table OS/2 in font")
	}
	name := table.Base().Fields().Name()
	if name != "OS/2" {
		t.Errorf(name)
	}
	x := table.Base().Fields().List().Get(1)
	t.Logf("x = %v", u16(x))
	if u16(x) != 400 {
		t.Errorf("expected xAvgCharWidth to be 400, is %d", u16(x))
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
	trace().Infof("========= loading done =================")
	return otf
}
