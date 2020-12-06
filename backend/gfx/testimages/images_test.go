package testimages

import (
	"testing"

	"github.com/npillmayer/schuko/gtrace"

	"github.com/npillmayer/arithm"
	"github.com/npillmayer/arithm/jhobby"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/backend/gfx"
	"github.com/npillmayer/tyse/backend/gfx/hobbyadapter"
)

func T() tracing.Trace {
	return gtrace.GraphicsTracer
}

func TestEmptyPath1(t *testing.T) {
	pic := gfx.NewPicture("empty", gfx.NewDebuggingSurface())
	pic.Shipout("test")
}

func TestPath1(t *testing.T) {
	pic := gfx.NewPicture("path1", gfx.NewDebuggingSurface())
	p, controls := jhobby.Nullpath().Knot(arithm.P(0, 0)).Curve().Knot(arithm.P(50, 50)).Curve().
		Knot(arithm.P(100, 65)).End()
	controls = jhobby.FindHobbyControls(p, controls)
	pic.Draw(hobbyadapter.Contour(p, controls))
	pic.Shipout("test")
}

func TestPath2(t *testing.T) {
	pic := gfx.NewPicture("path2", gfx.NewDebuggingSurface())
	p, controls := jhobby.Nullpath().Knot(arithm.P(10, 50)).Curve().Knot(arithm.P(50, 90)).Curve().
		Knot(arithm.P(90, 50)).End()
	controls = jhobby.FindHobbyControls(p, controls)
	pic.Draw(hobbyadapter.Contour(p, controls))
	pic.Shipout("test")
}

func TestPath3(t *testing.T) {
	pic := gfx.NewPicture("path3", gfx.NewDebuggingSurface())
	p, controls := jhobby.Nullpath().Knot(arithm.P(10, 50)).Curve().Knot(arithm.P(50, 90)).Curve().
		Knot(arithm.P(90, 50)).Curve().Knot(arithm.P(50, 10)).Curve().Cycle()
	controls = jhobby.FindHobbyControls(p, controls)
	pic.Draw(hobbyadapter.Contour(p, controls))
	pic.Shipout("test")
}
