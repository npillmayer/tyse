package frame

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/stretchr/testify/assert"
)

func TestBoxNullbox(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	//
	box := InitEmptyBox(&Box{})
	assert.Equal(t, box.Padding[Top], css.SomeDimen(0))
	assert.Equal(t, css.SomeDimen(0), box.BorderWidth[Right])
	assert.Equal(t, css.SomeDimen(0), box.Margins[Left])
	assert.Equal(t, box.W, css.DimenOption("auto"))
	assert.False(t, box.HasFixedBorderBoxWidth(true))
	assert.False(t, box.HasFixedBorderBoxHeight(true))
}

func TestFixContent1(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	//
	box := InitEmptyBox(&Box{})
	box.Padding[Left] = css.DimenOption("50%")
	box.FixContentWidth(60 * dimen.PT)
	assert.Equal(t, css.SomeDimen(60*dimen.PT), box.ContentWidth())
	box.Padding[Right] = css.DimenOption("10pt")
	assert.True(t, box.HasFixedBorderBoxWidth(false))
	t.Logf(box.DebugString())
	assert.Equal(t, css.SomeDimen(100*dimen.PT), box.BorderBoxWidth())
}

func TestFixContent2(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	//
	box := &Box{}
	box.FixContentWidth(60 * dimen.PT)
	assert.Equal(t, css.SomeDimen(60*dimen.PT), box.ContentWidth())
	assert.Equal(t, css.Dimen(), box.BorderBoxWidth())
	box.Padding[Left] = css.DimenOption("20pt")
	box.Padding[Right] = css.DimenOption("0")
	box.BorderWidth[Left] = css.DimenOption("0")
	box.BorderWidth[Right] = css.DimenOption("0")
	assert.Equal(t, css.SomeDimen(80*dimen.PT), box.BorderBoxWidth())
}

func TestFixContentBorderBoxSizing(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	//
	box := &Box{BorderBoxSizing: true}
	isFixed := box.FixContentWidth(60 * dimen.PT)
	assert.False(t, isFixed)
	assert.Equal(t, css.Dimen(), box.ContentWidth())

	box.Padding[Left] = css.DimenOption("10pt")
	box.Padding[Right] = css.DimenOption("10%")
	box.BorderWidth[Left] = css.DimenOption("0")
	box.BorderWidth[Right] = css.DimenOption("0")
	isFixed = box.FixContentWidth(80 * dimen.PT)
	t.Logf(box.DebugString())
	assert.Equal(t, css.SomeDimen(80*dimen.PT), box.ContentWidth())
	assert.Equal(t, css.SomeDimen(100*dimen.PT), box.BorderBoxWidth())
}

func TestFixBorderBoxWidth(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.EngineTracer.SetTraceLevel(tracing.LevelDebug)
	//
	box := InitEmptyBox(&Box{})
	box.Padding[Left] = css.DimenOption("10%")
	box.Padding[Right] = css.DimenOption("10%")
	box.FixBorderBoxWidth(120 * dimen.PT)
	t.Logf(box.DebugString())
	assert.Equal(t, css.SomeDimen(100*dimen.PT), box.ContentWidth())
	assert.Equal(t, css.SomeDimen(120*dimen.PT), box.BorderBoxWidth())
}

func TestFixBorderBoxBorderBoxSizing(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.EngineTracer.SetTraceLevel(tracing.LevelDebug)
	//
	box := InitEmptyBox(&Box{BorderBoxSizing: true})
	box.Padding[Left] = css.DimenOption("10%")
	box.Padding[Right] = css.DimenOption("10%")
	box.FixBorderBoxWidth(120 * dimen.PT)
	t.Logf(box.DebugString())
	assert.Equal(t, css.SomeDimen(96*dimen.PT), box.ContentWidth())
	assert.Equal(t, css.SomeDimen(120*dimen.PT), box.BorderBoxWidth())
}

func TestSetWidth(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.EngineTracer.SetTraceLevel(tracing.LevelDebug)
	//
	box := InitEmptyBox(&Box{})
	assert.True(t, box.W.Equals(css.Auto))
	box.Padding[Left] = css.DimenOption("10%")
	box.Padding[Right] = css.DimenOption("10%")
	box.SetWidth(css.SomeDimen(100 * dimen.PT))
	assert.False(t, box.HasFixedBorderBoxWidth(false))
	box.FixPrecentages(200 * dimen.PT)
	assert.True(t, box.HasFixedBorderBoxWidth(false))
	assert.True(t, box.HasFixedBorderBoxWidth(true))
	assert.Equal(t, 20*dimen.PT, box.Padding[Left].Unwrap())
	assert.Equal(t, css.SomeDimen(140*dimen.PT), box.TotalWidth())
}
