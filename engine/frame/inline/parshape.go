package inline

import (
	"fmt"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/khipu/linebreak"
)

func OutlineParshape(pbox *frame.PrincipalBox, leftAlign, rightAlign []*frame.Box) linebreak.ParShape {
	boundingBox := pbox.Box.Box // TODO check for nil
	polygon := paragraphPolygon(&boundingBox, leftAlign, rightAlign)
	T().Debugf("polygon = %v", polygon)
	if polygon == nil {
		return nil
	}
	return nil
}

type polygonParshape struct {
	lineskip dimen.Dimen
	width    dimen.Dimen
	polygon  *isoPolygon
}

// LineLength is part of interface ParShape. It returns the line width for line
// number l.
// TODO To calculate correctly, lineskip has to be variable. How?
func (pp polygonParshape) LineLength(l int32) dimen.Dimen {
	w := pp.width
	for _, b := range pp.polygon.stack {
		if int32(b.TopL.Y) > l*int32(pp.lineskip) {
			break
		}
		w = b.BotR.X - b.TopL.X
	}
	return w
}

type isoPolygon struct {
	stack []isoBox
}

func (iso isoPolygon) String() string {
	s := "P{\n"
	for _, b := range iso.stack {
		s += b.String() + "\n"
	}
	s += "}"
	return s
}

func paragraphPolygon(pbox *frame.Box, leftAlign, rightAlign []*frame.Box) *isoPolygon {
	parPolygon := &isoPolygon{
		stack: []isoBox{{ // inner box of paragraph's principal box
			TopL: pbox.TopL,
			BotR: pbox.BotR,
		}},
	}
	leftaligned := make([]isoBox, 0, len(leftAlign))
	for _, ch := range leftAlign {
		b := box2box(ch)
		leftaligned = insertBox(leftaligned, b)
	}
	rightaligned := make([]isoBox, 0, len(rightAlign))
	for _, ch := range rightAlign {
		b := box2box(ch)
		rightaligned = insertBox(rightaligned, b)
	}
	T().Debugf("leftaligned = %v", leftaligned)
	T().Debugf("rightaligned = %v", rightaligned)
	// now child boxes are ordered by X and grouped by alignment
	for _, b := range leftaligned {
		parPolygon = parPolygon.Subtract(b)
		T().Debugf("resulting polygon = %v", parPolygon)
		T().Debugf("===================")
	}
	for _, b := range rightaligned {
		parPolygon = parPolygon.Subtract(b)
		T().Debugf("resulting polygon = %v", parPolygon)
		T().Debugf("===================")
	}
	return parPolygon
}

func insertBox(a []isoBox, b isoBox) []isoBox {
	i := 0
	for _, bb := range a {
		if b.TopL.X <= bb.TopL.X {
			// insert b before bb
			break
		}
		i++
	}
	a = append(a[:i], append([]isoBox{b}, a[i:]...)...)
	return a
}

type isoBox struct {
	TopL dimen.Point
	BotR dimen.Point
}

var nullbox = isoBox{
	TopL: dimen.Point{X: 0, Y: 0},
	BotR: dimen.Point{X: 0, Y: 0},
}

func (b isoBox) String() string {
	return fmt.Sprintf("B[(%d,%d) (%d,%d)]", b.TopL.X, b.TopL.Y, b.BotR.X, b.BotR.Y)
}

// func polygonFromBox(f *frame.Box) *isoPolygon {
// 	iso := &isoPolygon{stack: make([]isoBox, 0, 1)}
// 	if f.FullWidth() > 0 {
// 		iso.stack = append(iso.stack, nullbox)
// 		iso.stack[0].TopL.X = f.TopL.X - f.Padding[frame.Left] - f.BorderWidth[frame.Left] -
// 			f.Margins[frame.Left]
// 		iso.stack[0].TopL.Y = f.TopL.Y - f.Padding[frame.Top] - f.BorderWidth[frame.Top] -
// 			f.Margins[frame.Top]
// 		iso.stack[0].BotR.X = f.BotR.X + f.Padding[frame.Left] + f.BorderWidth[frame.Left] +
// 			f.Margins[frame.Left]
// 		iso.stack[0].BotR.Y = f.BotR.Y + f.Padding[frame.Top] + f.BorderWidth[frame.Top] +
// 			f.Margins[frame.Top]
// 	}
// 	return iso
// }

func box2box(f *frame.Box) isoBox {
	b := isoBox{}
	b.TopL.X = f.TopL.X - f.Padding[frame.Left] - f.BorderWidth[frame.Left] -
		f.Margins[frame.Left]
	b.TopL.Y = f.TopL.Y - f.Padding[frame.Top] - f.BorderWidth[frame.Top] -
		f.Margins[frame.Top]
	b.BotR.X = f.BotR.X + f.Padding[frame.Left] + f.BorderWidth[frame.Left] +
		f.Margins[frame.Left]
	b.BotR.Y = f.BotR.Y + f.Padding[frame.Top] + f.BorderWidth[frame.Top] +
		f.Margins[frame.Top]
	return b
}

func intersect(box1, box2 isoBox) bool {
	if box2 == nullbox {
		return false
	}
	return !(box1.TopL.X >= box2.BotR.X ||
		box1.BotR.X <= box2.TopL.X ||
		box1.TopL.Y >= box2.BotR.Y ||
		box1.BotR.Y <= box2.TopL.Y)
}

func intersection(box1, box2 isoBox) isoBox {
	if !intersect(box1, box2) {
		return nullbox
	}
	intersec := isoBox{
		TopL: dimen.Point{
			X: max(box1.TopL.X, box2.TopL.X),
			Y: max(box1.TopL.Y, box2.TopL.Y),
		},
		BotR: dimen.Point{
			X: min(box1.BotR.X, box2.BotR.X),
			Y: min(box1.BotR.Y, box2.BotR.Y),
		},
	}
	return intersec
}

// Subtract takes away the area of a box from an iso polygon.
//
// I'm sure there are more elegant methods than to write out every
// sub-case, but it's late in the evening and I am too tired to be creative.
// For now, it lacks elegance, but it works.
//
func (iso *isoPolygon) Subtract(box isoBox) *isoPolygon {
	stk := make([]isoBox, 0, len(iso.stack)+1)
	T().Debugf("stack has length %d", len(iso.stack))
	for i, b := range iso.stack {
		if !intersect(b, box) {
			stk = append(stk, b)
			T().Debugf("box %v does not intersect stack box #%d = %v", box, i, b)
			T().Debugf("-X-----------------")
			continue
		}
		T().Debugf("box %v intersects with stack box #%d = %v", box, i, b)
		x := intersection(b, box)
		if x.TopL.X == b.TopL.X {
			if x.TopL.Y == b.TopL.Y {
				T().Debugf("(1)")
				top := isoBox{
					TopL: dimen.Point{X: x.BotR.X, Y: b.TopL.Y},
					BotR: dimen.Point{X: b.BotR.X, Y: x.BotR.Y},
				}
				bot := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: x.BotR.Y},
					BotR: dimen.Point{X: b.BotR.X, Y: b.BotR.Y},
				}
				stk = append(stk, top)
				if bot.TopL.Y < bot.BotR.Y {
					stk = append(stk, bot)
				}
			} else if x.BotR.Y == b.BotR.Y {
				T().Debugf("(2)")
				top := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: b.TopL.Y},
					BotR: dimen.Point{X: b.BotR.X, Y: x.TopL.Y},
				}
				bot := isoBox{
					TopL: dimen.Point{X: x.BotR.X, Y: x.TopL.Y},
					BotR: dimen.Point{X: b.BotR.X, Y: x.BotR.Y},
				}
				stk = append(stk, top)
				stk = append(stk, bot)
			} else {
				T().Debugf("(3)")
				// make 3 slices out of 1 box
				top := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: b.TopL.Y},
					BotR: dimen.Point{X: b.BotR.X, Y: x.TopL.Y},
				}
				mid := isoBox{
					TopL: dimen.Point{X: x.BotR.X, Y: x.TopL.Y},
					BotR: dimen.Point{X: b.BotR.X, Y: x.BotR.Y},
				}
				bot := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: x.BotR.Y},
					BotR: dimen.Point{X: b.BotR.X, Y: b.BotR.Y},
				}
				stk = append(stk, top)
				stk = append(stk, mid)
				stk = append(stk, bot)
			}
		} else if x.BotR.X == b.BotR.X {
			T().Debugf("(4)")
			if x.TopL.Y == b.TopL.Y {
				top := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: b.TopL.Y},
					BotR: dimen.Point{X: x.TopL.X, Y: x.BotR.Y},
				}
				bot := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: x.BotR.Y},
					BotR: dimen.Point{X: x.TopL.X, Y: b.BotR.Y},
				}
				stk = append(stk, top)
				if bot.TopL.Y < bot.BotR.Y {
					stk = append(stk, bot)
				}
			} else if x.BotR.Y == b.BotR.Y {
				T().Debugf("(5)")
				top := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: b.TopL.Y},
					BotR: dimen.Point{X: b.BotR.X, Y: x.TopL.Y},
				}
				bot := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: x.TopL.Y},
					BotR: dimen.Point{X: x.TopL.X, Y: b.BotR.Y},
				}
				stk = append(stk, top)
				stk = append(stk, bot)
			} else {
				T().Debugf("(6)")
				// make 3 slices out of 1 box
				top := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: b.TopL.Y},
					BotR: dimen.Point{X: b.BotR.X, Y: x.TopL.Y},
				}
				mid := isoBox{
					TopL: dimen.Point{X: b.BotR.X, Y: x.TopL.Y},
					BotR: dimen.Point{X: x.TopL.X, Y: x.BotR.Y},
				}
				bot := isoBox{
					TopL: dimen.Point{X: b.TopL.X, Y: x.BotR.Y},
					BotR: dimen.Point{X: x.TopL.X, Y: b.BotR.Y},
				}
				stk = append(stk, top)
				stk = append(stk, mid)
				stk = append(stk, bot)
			}
		} else {
			panic(fmt.Sprintf("cannot cut a hole, x = %v", x))
		}
		T().Debugf("-------------------")
	}
	return &isoPolygon{stack: stk}
}

// --- Helpers ----------------------------------------------------------

func min(a, b dimen.Dimen) dimen.Dimen {
	if a < b {
		return a
	}
	return b
}

func max(a, b dimen.Dimen) dimen.Dimen {
	if a > b {
		return a
	}
	return b
}
