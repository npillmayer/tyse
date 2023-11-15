package css_test

import (
	"testing"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/percent"
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/dom/style/css"
)

func TestDimenBasic(t *testing.T) {
	ten := css.JustDimen(dimen.PT * 10)
	var du dimen.DU
	switch m := ten.Match(); m {
	case m.Just(&du):
		t.Logf("du = %s", du)
	default:
		t.Errorf("expected Just(10pt) to be a fixed value, isn't: %#v", ten)
	}

	auto := css.Auto()
	switch m := auto.Match(); m {
	case m.IsKind(css.Auto()):
		t.Logf("dimen is auto")
	default:
		t.Errorf("expected dimen auto to match auto, isn't: %#v", auto)
	}

	pcnt := css.Percentage(percent.FromInt(80))
	var p percent.Percent
	switch m := pcnt.Match(); m {
	case m.Percentage(&p):
		t.Logf("percent = %s", p)
	default:
		t.Errorf("expected Percentage(80) to be a percentage value, isn't: %#v", pcnt)
	}
}

func TestDimenPattern(t *testing.T) {
	ten := css.JustDimen(dimen.PT * 10)
	// now use it
	var du dimen.DU
	m := css.DimenPattern[int](ten)
	zehn := m.OneOf(css.DimenPatterns[int]{
		Just:    10,
		Auto:    0,
		Default: -1,
	})
	if zehn != 10 {
		t.Errorf("expected zehn == 10, isn't: %#v", zehn)
	}

	d := css.JustDimen(dimen.PT * 10)
	// now use it
	e := css.DimenPattern[dimen.DU](d)
	distance := e.OneOf(css.DimenPatterns[dimen.DU]{
		Just:    e.With(&du).Const(2 * du),
		Auto:    0,
		Default: -1,
	})
	if distance != 2*10*dimen.PT {
		t.Errorf("expected distance to be %v, isn't: %#v", 20*dimen.PT, distance)
	}
}

func TestDimenAuto(t *testing.T) {
	p := style.Property("auto")
	d := css.DimenOption(p)
	m := css.DimenPattern[string](d)
	x := m.OneOf(css.DimenPatterns[string]{
		Unset:   "NONE",
		Auto:    "AUTO",
		Default: "NONE",
	})
	if x != "AUTO" {
		t.Errorf("expected AUTO, have %v", x)
	}
}
