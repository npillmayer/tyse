package css_test

import (
	"testing"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/dom/style/css"
)

func TestPositionBasic(t *testing.T) {
	a := css.Absolute(nil)
	var o []css.PositionOffset
	switch m := a.Match(); m {
	case m.Absolute(&o):
		t.Logf("offsets = %v", o)
	default:
		t.Errorf("expected Absolute() to be an absolute position, isn't: %#v", a)
	}

	static := css.Static()
	switch m := static.Match(); m {
	case m.IsKind(css.Static()):
		t.Logf("position is static")
	default:
		t.Errorf("expected position to match kind(static), isn't: %#v", static)
	}
}

func TestPositionPattern(t *testing.T) {
	o := []css.PositionOffset{
		{css.JustDimen(10 * dimen.PT), css.Bottom},
	}
	f := css.Fixed(o)
	// now use it
	m := css.PositionPattern[int](f)
	out := m.OneOf(css.PositionPatterns[int]{
		Unset:   10,
		Fixed:   99,
		Default: -1,
	})
	if out != 99 {
		t.Errorf("expected out to be 99, isn't: %#v", out)
	}

	e := css.PositionPattern[[]css.PositionOffset](f)
	off := e.OneOf(css.PositionPatterns[[]css.PositionOffset]{
		Fixed:    e.With(&o).Const(o),
		Relative: css.ZeroOffsets(),
		Default:  css.ZeroOffsets(),
	})
	t.Logf("offsets = %v", off)
	if len(off) != 4 {
		t.Errorf("expected 4 offsets, aren't: %#v", off)
	}
}

func TestPositionAuto(t *testing.T) {
	p := style.Property("absolute")
	pos := css.Position(p)
	m := css.PositionPattern[string](pos)
	x := m.OneOf(css.PositionPatterns[string]{
		Unset:    "NONE",
		Absolute: "ABSOLUTE",
		Default:  "NONE",
	})
	if x != "ABSOLUTE" {
		t.Errorf("expected ABSOLUTE, have %v", x)
	}
}
