package inline

import (
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/parameters"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
	"github.com/npillmayer/tyse/engine/frame/khipu"
	"github.com/npillmayer/tyse/engine/frame/khipu/linebreak"
	"github.com/npillmayer/tyse/engine/frame/khipu/linebreak/firstfit"
	"github.com/npillmayer/tyse/engine/glyphing/monospace"
	"github.com/npillmayer/tyse/engine/tree"
)

// --- Line Boxes ------------------------------------------------------------

// TypeLine and TypeParagraph are additional container types defined in this package.
const (
	TypeLine      frame.ContainerType = 200
	TypeParagraph frame.ContainerType = 201
)

// LineBox is a type for CSS inline text boxes.
type LineBox struct {
	frame.ContainerBase
	//tree.Node
	Box     *frame.Box
	khipu   *khipu.Khipu
	indent  dimen.DU      // horizontal offset of the text within the line box
	pos     int64         // start position within the khipu
	length  int64         // length of the segment for this line
	context frame.Context // formatting context
	//ChildInx uint32      // this box represents a text node at #ChildInx of the principal box
}

func NewLineBox(k *khipu.Khipu, start, length int64, indent dimen.DU) *LineBox {
	lbox := &LineBox{
		Box:    frame.InitEmptyBox(&frame.Box{}),
		khipu:  k,
		pos:    start,
		length: length,
		indent: indent,
	}
	lbox.Box.H = css.SomeDimen(12 * dimen.PT)
	lbox.Payload = lbox
	return lbox
}

// DOMNode returns the underlying DOM node for a render tree element.
// For line boxes, it returns the DOM node corresponding to the parent container,
// which should be of type PrincipalBox.
func (lbox *LineBox) DOMNode() *dom.W3CNode {
	parent := boxtree.TreeNodeAsPrincipalBox(lbox.Parent())
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

// Type returns TypeLine
func (pbox *LineBox) Type() frame.ContainerType {
	return TypeLine
}

// IsAnonymous will always return true for a line box.
// func (lbox *LineBox) IsAnonymous() bool {
// 	return false
// }

// IsText will always return false for a line box.
// func (lbox *LineBox) IsText() bool {
// 	return false
// }

// DisplayMode always returns inline.
// func (lbox *LineBox) DisplayMode() frame.DisplayMode {
// 	return frame.InlineMode
// }

// ChildIndices returns 0, 0.
// func (lbox *LineBox) ChildIndices() (uint32, uint32) {
// 	return 0, 0
// }

func (lbox *LineBox) Context() frame.Context {
	return nil
}

func (lbox *LineBox) SetContext(ctx frame.Context) {
	lbox.context = ctx
}

func (lbox *LineBox) PresetContained() bool {
	panic("TODO")
	return false
}

func (lbox *LineBox) AppendToPrincipalBox(pbox *boxtree.PrincipalBox) {
	//boxtree.Inline(pbox.Context()).AddLineBox(lbox)
	panic("TODO ?")
}

var _ frame.RenderTreeNode = &LineBox{}

// --- Breaking paragraphs into lines ----------------------------------------

// BreakParagraph breaks a khipu (of a paragraph) into lines, given the outline
// of the frame where the text has to fit into.
// It returns the same pbox, but now including anonymous line boxes for the text,
// and the height value of the principal box will be set.
//
// If an error occurs during line-breaking, a pbox of nil is returned, together with the
// error value.
//
func BreakParagraph(para *Paragraph, box *frame.Box) ([]*frame.ContainerBase, error) {
	//
	// TODO
	// find all children with align=left or align=right and collect their boxes
	// there should be an API for this in frame/layout.
	//
	//leftAlign, rightAlign := collectFloatBoxes(pbox)
	//parshape := OutlineParshape(pbox, leftAlign, rightAlign)
	parshape := OutlineParshape(box, nil, nil)
	if parshape == nil {
		tracer().Errorf("could not create a parshape for principal box")
	}
	cursor := linebreak.NewFixedWidthCursor(khipu.NewCursor(para.Khipu), 10*dimen.BP, 0)
	breakpoints, err := firstfit.BreakParagraph(cursor, parshape, nil)
	if err != nil {
		return nil, err
	}
	tracer().Debugf("text broken up with %d breaks: %v", len(breakpoints), breakpoints)
	//
	// TODO
	// assemble the broken line segments into anonymous line boxes
	tracer().Debugf("     |---------+---------+---------+---------+---------50--------|")
	j := int64(0)
	var lines []*frame.ContainerBase
	for i := 1; i < len(breakpoints); i++ {
		pos := breakpoints[i].Position()
		tracer().Debugf("%3d: %s", i, para.Khipu.Text(j, pos))
		l := pos - j
		indent := dimen.DU(0) // TODO derive from parshape
		linebox := NewLineBox(para.Khipu, breakpoints[i].Position(), l, indent)
		linebox.Box.W = box.W
		lines = append(lines, &linebox.ContainerBase)
		//linebox.AppendToPrincipalBox(pbox)
		j = breakpoints[i].Position()
	}
	//
	return lines, nil
}

func EncodeTextOfParagraph(c *frame.ContainerBase) (*Paragraph, []*frame.ContainerBase, error) {
	paraText, blocks, err := paragraphTextFromBox(c)
	if err != nil {
		tracer().Errorf(err.Error())
		return nil, []*frame.ContainerBase{}, err
	}
	paraText.Regs = parameters.NewTypesettingRegisters()
	paraText.Regs = adaptTypesettingRegisters(paraText.Regs, c)
	paraText.Khipu, err = khipu.EncodeParagraph(paraText.Paragraph, 0,
		monospace.Shaper(11*dimen.PT, nil), nil, paraText.Regs)
	if err != nil || paraText.Khipu == nil {
		tracer().Errorf("lines: khipu resulting from paragraph is nil")
		return nil, []*frame.ContainerBase{}, err
	}
	return paraText, blocks, err
}

func XFindParaWidthAndText(pbox *ParagraphBox, rootctx frame.Context) (
	[]*frame.ContainerBase, frame.Context, error) {
	//
	paraText, blocks, err := paragraphTextFromBox(&pbox.ContainerBase)
	if err != nil {
		tracer().Errorf(err.Error())
		return []*frame.ContainerBase{}, rootctx, err
	}
	regs := parameters.NewTypesettingRegisters()
	regs = adaptTypesettingRegisters(regs, &pbox.ContainerBase)
	k, err := khipu.EncodeParagraph(paraText.Paragraph, 0, monospace.Shaper(11*dimen.PT, nil), nil, regs)
	if err != nil || k == nil {
		tracer().Errorf("lines: khipu resulting from paragraph is nil")
		return []*frame.ContainerBase{}, rootctx, err
	}
	pbox.para = paraText
	pbox.khipu = k
	//
	// if ctx := boxtree.CreateContextForContainer(pbox, false); ctx != nil {
	// we have to respect floats from the root context
	// ...
	// then move on to the paragraphs context
	// rootctx = ctx
	// }
	if !pbox.Box.W.IsAbsolute() {
		panic("width of paragraph's box not known")
	}
	// if pbox.Box.Width() == 0 {
	// 	pbox.Box.BotR = dimen.Point{
	// 		X: pbox.Box.TopL.X + 60*10*dimen.BP,
	// 		Y: dimen.Infinity,
	// 	}
	// }
	return blocks, rootctx, nil
	/*
		pbox, err = BreakParagraph(k, pbox, regs)
		if err != nil {
			T().Errorf(err.Error())
			return err
		}
		if pbox.Box.BotR.Y == dimen.Infinity {
			pbox.Box.BotR.Y = addLineHeights(pbox)
		}
		return nil
	*/
}

func adaptTypesettingRegisters(regs *parameters.TypesettingRegisters, c *frame.ContainerBase) *parameters.TypesettingRegisters {
	return regs
}

func addLineHeights(pbox *boxtree.PrincipalBox) dimen.DU {
	var height dimen.DU
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

func collectFloatBoxes(pbox *boxtree.PrincipalBox) ([]*frame.Box, []*frame.Box) {
	// TODO
	// Float handling has to be re-thought completely
	return []*frame.Box{}, []*frame.Box{}
}

func Layout(c frame.ContainerInterf) {
	tracer().Debugf("Layout of sub-block")
	if c.DisplayMode().Inner().Contains(css.InlineMode) {
		// call layout paragraph
	} else {
		// call layout block
	}
	// other cases
}
