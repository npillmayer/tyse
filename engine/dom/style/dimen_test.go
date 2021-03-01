package style

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/option"
)

func TestDimen(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	p := Property("100pt")
	d := p.DimenOption()
	if d.Unwrap() != dimen.Dimen(100)*dimen.PT {
		t.Errorf("expected 100 PT (%d), have %d", 100*dimen.PT, d)
	}
	//
	p = Property("auto")
	d = p.DimenOption()
	x, err := d.Match(option.Of{
		option.None: "NONE",
		Auto:        "AUTO",
	})
	if err != nil || x != "AaUTO" {
		t.Errorf("expected AUTO, have %v with error %v", x, err)
	}
}
