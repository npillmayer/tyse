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
	t.Logf("list is %s of length %v", lang.Name(), len(langlist))
	if lang.Name() != "LangSys" || len(langlist) != 10 {
		t.Errorf("expected LangSys[IPPH] to contain 10 feature entries, has %d", len(langlist))
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
		t.Errorf("expected table have name 'name', have %s", name)
	}
	key := MakeTag([]byte{0, 0, 0, 1})
	x := table.Base().Fields().Map().Lookup(key).Navigate().Name()
	t.Logf("x = %v", x)
	t.Fail()
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
