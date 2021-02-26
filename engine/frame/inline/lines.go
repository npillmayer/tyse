package inline

import (
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/parameters"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/khipu"
	"github.com/npillmayer/tyse/engine/frame/khipu/linebreak"
	"github.com/npillmayer/tyse/engine/frame/khipu/linebreak/firstfit"
	"github.com/npillmayer/tyse/engine/glyphing/monospace"
	"github.com/npillmayer/tyse/engine/tree"
)

// --- Line Boxes ------------------------------------------------------------

// LineBox is a type for CSS inline text boxes.
type LineBox struct {
	tree.Node
	Box      *frame.Box
	khipu    *khipu.Khipu
	indent   dimen.Dimen // horizontal offset of the text within the line box
	pos      int64       // start position within the khipu
	length   int64       // length of the segment for this line
	ChildInx uint32      // this box represents a text node at #ChildInx of the principal box
}

func NewLineBox(k *khipu.Khipu, start, length int64, indent dimen.Dimen) *LineBox {
	lbox := &LineBox{
		khipu:  k,
		pos:    start,
		length: length,
		indent: indent,
	}
	lbox.Payload = lbox
	return lbox
}

// DOMNode returns the underlying DOM node for a render tree element.
// For line boxes, it returns the DOM node corresponding to the parent container,
// which should be of type PrincipalBox.
func (lbox *LineBox) DOMNode() *dom.W3CNode {
	parent := frame.TreeNodeAsPrincipalBox(lbox.Parent())
	if parent == nil {
		return nil
	}
	return parent.DOMNode()
}

// TreeNode returns the underlying tree node for a line box.
func (lbox *LineBox) TreeNode() *tree.Node {
	return &lbox.Node
}

// CSSBox returns the underlying box of a line.
func (lbox *LineBox) CSSBox() *frame.Box {
	return lbox.Box
}

// IsAnonymous will always return true for a line box.
func (lbox *LineBox) IsAnonymous() bool {
	return false
}

// IsText will always return false for a line box.
func (lbox *LineBox) IsText() bool {
	return false
}

// DisplayMode always returns inline.
func (lbox *LineBox) DisplayMode() frame.DisplayMode {
	return frame.InlineMode
}

// ChildIndices returns 0, 0.
func (lbox *LineBox) ChildIndices() (uint32, uint32) {
	return 0, 0
}

func (lbox *LineBox) Context() frame.Context {
	return nil
}

func (lbox *LineBox) AppendToPrincipalBox(pbox *frame.PrincipalBox) {
	frame.Inline(pbox.Context()).AddLineBox(lbox)
}

var _ frame.Container = &LineBox{}

// --- Breaking paragraphs into lines ----------------------------------------

// BreakParagraph breaks a khipu (of a paragraph) into lines, given the outline
// of the frame where the text has to fit into.
// It returns the same pbox, but now including anonymous line boxes for the text,
// and the height value of the principal box will be set.
//
// If an error occurs during line-breaking, a pbox of nil is returned, together with the
// error value.
//
func BreakParagraph(k *khipu.Khipu, pbox *frame.PrincipalBox,
	regs *parameters.TypesettingRegisters) (*frame.PrincipalBox, error) {
	//
	// TODO
	// find all children with align=left or align=right and collect their boxes
	// there should be an API for this in frame/layout.
	//
	leftAlign, rightAlign := collectAlignedBoxes(pbox)
	parshape := OutlineParshape(pbox, leftAlign, rightAlign)
	if parshape == nil {
		T().Errorf("could not create a parshape for principal box")
	}
	cursor := linebreak.NewFixedWidthCursor(khipu.NewCursor(k), 10*dimen.BP, 0)
	breakpoints, err := firstfit.BreakParagraph(cursor, parshape, nil)
	if err != nil {
		return nil, err
	}
	T().Debugf("text broken up into %d lines", len(breakpoints))
	//
	// TODO
	// assemble the broken line segments into anonymous line boxes
	T().Debugf("     |---------+---------+---------+---------+---------50--------|")
	j := int64(0)
	for i := 1; i < len(breakpoints); i++ {
		pos := breakpoints[i].Position()
		T().Debugf("%3d: %s", i, k.Text(j, pos))
		l := pos - j
		indent := dimen.Dimen(0) // TODO derive from parshape
		linebox := NewLineBox(k, breakpoints[i].Position(), l, indent)
		linebox.AppendToPrincipalBox(pbox)
		j = breakpoints[i].Position()
	}
	//
	return nil, nil
}

func PositionLines(pbox *frame.PrincipalBox) (*frame.PrincipalBox, error) {
	paraText, err := ParagraphFromBox(pbox)
	if err != nil {
		T().Errorf(err.Error())
		return nil, err
	}
	regs := parameters.NewTypesettingRegisters()
	regs = adaptTypesettingRegisters(regs, pbox)
	k, err := khipu.EncodeParagraph(paraText.Paragraph, 0, monospace.Shaper(11*dimen.PT, nil), nil, regs)
	if err != nil || k == nil {
		T().Errorf("lines: khipu resulting from paragraph is nil")
		return nil, err
	}
	if pbox.Box.Width() == 0 {
		pbox.Box.BotR = dimen.Point{
			X: pbox.Box.TopL.X + 60*10*dimen.BP,
			Y: dimen.Infinity,
		}
	}
	pbox, err = BreakParagraph(k, pbox, regs)
	if err != nil {
		T().Errorf(err.Error())
		return nil, err
	}
	if pbox.Box.BotR.Y == dimen.Infinity {
		pbox.Box.BotR.Y = addLineHeights(pbox)
	}
	return pbox, nil
}

func adaptTypesettingRegisters(regs *parameters.TypesettingRegisters, pbox *frame.PrincipalBox) *parameters.TypesettingRegisters {
	return regs
}

func addLineHeights(pbox *frame.PrincipalBox) dimen.Dimen {
	var height dimen.Dimen
	// ctx := pbox.Context()
	// children := ctx.TreeNode().Children()
	// var lastLine *frame.Box
	// for _, ch := range children {
	// 	if c, ok := ch.Payload.(frame.Container); ok {
	// 		height += c.CSSBox().Height()
	// 		_, smallerMargin := frame.CollapseMargins(lastLine, c.CSSBox())
	// 		height -= smallerMargin
	// 		lastLine = c.CSSBox()
	// 	}
	// }
	return height
}

func collectAlignedBoxes(pbox *frame.PrincipalBox) ([]*frame.Box, []*frame.Box) {
	// TODO
	// Float handling has to be re-thought completely
	return []*frame.Box{}, []*frame.Box{}
}
