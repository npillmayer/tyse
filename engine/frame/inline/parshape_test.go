package inline

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/frame"
)

func TestBoxIntersection(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	//
	box1 := makeIsoBox(0, 0, 20, 30)
	box2 := makeIsoBox(10, 10, 50, 50)
	intersec := intersection(box1, box2)
	t.Logf("intersection box = %v", intersec)
	if intersec.TopL.X != 10 || intersec.TopL.Y != 10 {
		t.Errorf("expected intersection to have upper left corner of (10,10), have %v", intersec.TopL)
	}
	if intersec.BotR.X != 20 || intersec.BotR.Y != 30 {
		t.Errorf("expected intersection to have lower right corner of (20,30), have %v", intersec.BotR)
	}

	box1 = makeIsoBox(0, 0, 100, 20)
	box2 = makeIsoBox(200, 20, 500, 50)
	x := intersect(box1, box2)
	if x {
		t.Errorf("boxes do not intersect, but are reported to do so")
	}
}

func TestParaPolygon(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.EngineTracer.SetTraceLevel(tracing.LevelDebug)
	//
	para, leftAlign, rightAlign := makePara()
	T().Debugf("leftAlign = %v", leftAlign)
	_ = OutlineParshape(para, leftAlign, rightAlign)
	t.Fail()
}

// --- Helpers ----------------------------------------------------------

func makeIsoBox(a, b, c, d dimen.Dimen) isoBox {
	return isoBox{
		TopL: dimen.Point{X: a, Y: b},
		BotR: dimen.Point{X: c, Y: d},
	}
}

func makePara() (*frame.PrincipalBox, []*frame.Box, []*frame.Box) {
	para := frame.NewPrincipalBox(nil, frame.BlockMode, frame.InlineMode)
	para.Box = &frame.StyledBox{}
	para.Box.BotR = dimen.Point{X: 500, Y: 800}
	//
	lalgn1 := frame.Box{}
	lalgn1.TopL = dimen.Point{X: 0, Y: 0}
	lalgn1.BotR = dimen.Point{X: 100, Y: 20}
	//
	lalgn2 := frame.Box{}
	lalgn2.TopL = dimen.Point{X: 0, Y: 20}
	lalgn2.BotR = dimen.Point{X: 200, Y: 50}
	//
	ralgn := frame.Box{}
	ralgn.TopL = dimen.Point{X: 300, Y: 500}
	ralgn.BotR = dimen.Point{X: 500, Y: 800}
	//
	return para, []*frame.Box{&lalgn1, &lalgn2}, []*frame.Box{&ralgn}
}
