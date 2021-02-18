package inline

import (
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/parameters"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/khipu"
	"github.com/npillmayer/tyse/engine/khipu/linebreak"
	"github.com/npillmayer/tyse/engine/khipu/linebreak/firstfit"
)

// BreakParagraph breaks a khipu (of a paragraph) into lines, given the outline
// of the frame where the text has to fit into.
// It returns the same pbox, but now including anonymous line boxes for the text,
// and the height value of the principal box will be set.
//
// If an error occurs during line-breaking, a pbox of nil is returned, together with the
// error value.
//
func BreakParagraph(k *khipu.Khipu, pbox *frame.PrincipalBox,
	params *parameters.TypesettingParameter) (*frame.PrincipalBox, error) {
	//
	// TODO
	// find all children with align=left or align=right and collect their boxes
	// there should be an API for this in frame/layout.
	//
	var leftAlign, rightAlign []*frame.Box
	parshape := OutlineParshape(pbox, leftAlign, rightAlign)
	cursor := linebreak.NewFixedWidthCursor(khipu.NewCursor(k), 10*dimen.BP, 0)
	breakpoints, err := firstfit.BreakParagraph(cursor, parshape, nil)
	if err != nil {
		return nil, err
	}
	T().Debugf("text broken up into %d lines", len(breakpoints))
	//
	// TODO
	// assemble the broken line segments into anonymous line boxes
	j := int64(0)
	for i := 1; i < len(breakpoints); i++ {
		pos := breakpoints[i].Position()
		T().Debugf("%3d: %s", i, k.Text(j, pos))
		l := pos - j
		indent := dimen.Dimen(0) // TODO derive from parshape
		linebox := frame.NewLineBox(k, breakpoints[i].Position(), l, indent)
		pbox.AppendLineBox(linebox)
		j = breakpoints[i].Position()
	}
	//
	return nil, nil
}
