package layout

import (
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
	"github.com/npillmayer/tyse/engine/frame/inline"
)

const (
	TypeBlockFormattingContext  frame.FormattingContextType = 100
	TypeInlineFormattingContext frame.FormattingContextType = 101
)

// --- Block Formatting Context ----------------------------------------------

// https://developer.mozilla.org/en-US/docs/Web/Guide/CSS/Block_formatting_context

// BlockContext establishes a CSS block formatting context.
//
// “Block-level boxes are boxes that participate in a block formatting context.
// Each block-level element generates a principal block-level box that contains
// descendant boxes and generated content and is also the box involved in any
// positioning scheme. Some block-level elements may generate additional boxes
// in addition to the principal box [for example,]: 'list-item' elements. These
// additional boxes are placed with respect to the principal box.”
//
// A new BFC will behave much like the outermost document in that it becomes a
// mini-layout inside the main layout. A BFC contains everything inside it,
// float and clear only apply to items inside the same formatting context,
// and margins only collapse between elements in the same formatting context.
//
type BlockContext struct {
	frame.ContextBase
}

func NewBlockContext(c frame.Container, isRoot bool) *BlockContext {
	ctx := &BlockContext{}
	ctx.IsRootCtx = isRoot
	ctx.C = c
	ctx.Payload = ctx
	return ctx
}

func Block(ctx frame.Context) *BlockContext {
	if block, ok := ctx.(*BlockContext); ok {
		return block
	}
	panic("context is not a block context")
}

func (ctx *BlockContext) Type() frame.FormattingContextType {
	return TypeBlockFormattingContext
}

func (ctx *BlockContext) AddContained(c frame.Container) {
	if c.DisplayMode().Outer() == css.InlineMode {
		anon := boxtree.NewAnonymousBox(css.BlockMode | css.InnerInlineMode)
		c.TreeNode().Isolate()
		anon.AddChild(c.TreeNode())
		ctx.AddChild(anon.TreeNode())
		T().Debugf("block context added [%v] wrapped in anon box", c.DOMNode().NodeName())
		return
	}
	// if c.DOMNode().ComputedStyles().GetPropertyValue("float") != style.NullStyle {
	// 	T().P("context", "block").Errorf("float box cannot be added")
	// 	panic("illegal argument for BlockContext.AddBox(c)")
	// }
	// if c.DOMNode().ComputedStyles().GetPropertyValue("position") == "absolute" ||
	// 	c.DOMNode().ComputedStyles().GetPropertyValue("position") == "fixed" {
	// 	//
	// 	T().P("context", "block").Errorf("child container has absolute or fixed position")
	// 	panic("illegal argument for BlockContext.AddBox(c)")
	// }
	c.TreeNode().Isolate()
	if ctx.C.TreeNode().IndexOfChild(c.TreeNode()) >= 0 {
		panic("container is child container; cannot have 2 parents")
	}
	T().Debugf("block context added [%v]", c.DOMNode().NodeName())
	ctx.AddChild(c.TreeNode())
}

func (ctx *BlockContext) Layout(flowRoot *frame.FlowRoot) error {
	H := dimen.Zero
	for _, c := range ctx.Contained() {
		T().Debugf("[%s] positions box [%s]", boxtree.ContainerName(ctx.Container()),
			boxtree.ContainerName(c))
		c.CSSBox().TopL.Y = H
		if !c.CSSBox().H.IsAbsolute() {
			return ErrHeightNotFixed
		}
		H += c.CSSBox().H.Unwrap()
	}
	ctx.Container().CSSBox().H = css.SomeDimen(H)
	return nil
}

func (ctx *BlockContext) Measure() (frame.Size, css.DimenT, css.DimenT) {
	wmax, h, margin := dimen.Zero, dimen.Zero, dimen.Zero
	children := ctx.Contained()
	for i, ch := range children {
		chw, chh := ch.CSSBox().TotalWidth(), ch.CSSBox().TotalHeight()
		if !chw.IsAbsolute() || !chh.IsAbsolute() {
			return frame.Size{}, css.Dimen(), css.Dimen()
		}
		if w := ch.CSSBox().W.Unwrap(); w > wmax {
			wmax = w
		}
		h += ch.CSSBox().H.Unwrap()
		if i == 0 && !ctx.IsFlowRoot() {
			h -= ch.CSSBox().Margins[frame.Top].Unwrap()
		} else if i > 0 {
			minMargin := dimen.Min(margin, ch.CSSBox().Margins[frame.Top].Unwrap())
			h -= minMargin
		}
		margin = ch.CSSBox().Margins[frame.Bottom].Unwrap()
		if i == len(children) && !ctx.IsFlowRoot() {
			h -= margin
		}
	}
	//return css.SomeDimen(wmax), css.SomeDimen(h)
	return ctx.Container().CSSBox().Size, css.SomeDimen(0), css.SomeDimen(0)
}

var _ frame.Context = &BlockContext{}

// --- Inline Context --------------------------------------------------------

type InlineContext struct {
	frame.ContextBase
	lines []frame.Container
}

func NewInlineContext(c frame.Container, isRoot bool) *InlineContext {
	ctx := &InlineContext{}
	ctx.IsRootCtx = isRoot
	ctx.C = c
	ctx.Payload = ctx
	return ctx
}

func Inline(ctx frame.Context) *InlineContext {
	if inline, ok := ctx.(*InlineContext); ok {
		return inline
	}
	panic("context is not an inline context")
}

func (ctx *InlineContext) Type() frame.FormattingContextType {
	return TypeInlineFormattingContext
}

func (ctx *InlineContext) AddContained(c frame.Container) {
	if c.DisplayMode().Outer() == css.BlockMode {
		ctx.AddChild(c.TreeNode())
		T().Debugf("inline context added block [%v]", c.DOMNode().NodeName())
		return
	} else if c.DisplayMode().Outer() != css.InlineMode {
		anon := boxtree.NewAnonymousBox(css.InlineMode | css.InnerInlineMode)
		anon.CSSBox().W = css.DimenOption(style.Property("fit-content"))
		c.TreeNode().Isolate()
		anon.AddChild(c.TreeNode())
		ctx.AddChild(anon.TreeNode())
		T().Debugf("inline context added [%v] wrapped in anon box", c.DOMNode().NodeName())
		return
	}
	c.TreeNode().Isolate()
	if ctx.C.TreeNode().IndexOfChild(c.TreeNode()) >= 0 {
		panic("container is child container; cannot have 2 parents")
	}
	T().Debugf("inline context added [%v]", c.DOMNode().NodeName())
	T().Debugf("text = '%s'", c.DOMNode().HTMLNode().Data)
	ctx.AddChild(c.TreeNode())
}

func (ctx *InlineContext) addLines(lines ...frame.Container) {
	boxcnt := len(ctx.Contained())
	//size, _, mBot := ctx.Measure()
	size := ctx.Container().CSSBox().Size
	h, margin := dimen.Zero, dimen.Zero
	if !size.H.IsNone() {
		h = size.H.Unwrap()
		if lastbox := lastbox(ctx); lastbox != nil {
			margin = lastbox.Margins[frame.Bottom].Unwrap()
		}
	}
	for i, line := range lines {
		lh := line.CSSBox().TotalHeight()
		if !lh.IsAbsolute() {
			panic("line box must have fixed height")
		}
		h += lh.Unwrap()
		if boxcnt == 0 && !ctx.IsFlowRoot() {
			h -= line.CSSBox().Margins[frame.Top].Unwrap()
		} else if boxcnt > 0 {
			minMargin := dimen.Min(margin, line.CSSBox().Margins[frame.Top].Unwrap())
			h -= minMargin
		}
		line.CSSBox().TopL.Y = h
		ctx.lines = append(ctx.lines, line)
		margin = line.CSSBox().Margins[frame.Bottom].Unwrap()
		if i == len(ctx.lines) && !ctx.IsFlowRoot() {
			h -= margin
		}
		boxcnt++
	}
}

func (ctx *InlineContext) Layout(flowRoot *frame.FlowRoot) error {
	para, blocks, err := inline.TextOfParagraph(ctx.Container())
	if err != nil {
		return err
	} else if len(blocks) > 0 {
		T().Debugf("layout of inline container: %d enclosed blocks", len(blocks))
	}
	box := ctx.Container().CSSBox()
	lines, err := inline.BreakParagraph(para, box)
	if err != nil {
		return err
	}
	T().Debugf("paragraph broken into %d lines", len(lines))
	if len(lines) > 0 {
		last := lines[len(lines)-1]
		ctx.Container().CSSBox().H = css.SomeDimen(last.CSSBox().TopL.Y + last.CSSBox().H.Unwrap())
		ctx.addLines(lines...)
	}
	return nil
}

func (ctx *InlineContext) Measure() (frame.Size, css.DimenT, css.DimenT) {
	// h, margin := dimen.Zero, dimen.Zero
	// for i, line := range ctx.lines {
	// 	lh := line.CSSBox().TotalHeight()
	// 	if !lh.IsAbsolute() {
	// 		return frame.Size{}, css.Dimen(), css.Dimen()
	// 	}
	// 	//h += line.CSSBox().H.Unwrap()
	// 	h += lh.Unwrap()
	// 	if i == 0 && !ctx.IsFlowRoot() {
	// 		h -= line.CSSBox().Margins[frame.Top].Unwrap()
	// 	} else if i > 0 {
	// 		minMargin := dimen.Min(margin, line.CSSBox().Margins[frame.Top].Unwrap())
	// 		h -= minMargin
	// 	}
	// 	margin = line.CSSBox().Margins[frame.Bottom].Unwrap()
	// 	if i == len(ctx.lines) && !ctx.IsFlowRoot() {
	// 		h -= margin
	// 	}
	// }
	//
	return ctx.Container().CSSBox().OuterBox().Size, css.SomeDimen(0), css.SomeDimen(0)
}

// ---------------------------------------------------------------------------

/*
A new root (block) formatting context is created by:

 *  The root element of the document (<html>).
 *  Floats (elements where float isn't none).
 *  Absolutely positioned elements (elements where position is absolute or fixed).
 *  Inline-blocks (elements with display: inline-block).
 *  Table cells (elements with display: table-cell, which is the default for HTML table cells).
 *  Table captions (elements with display: table-caption, which is the default for HTML table captions).
 *  Anonymous table cells implicitly created by the elements with display: table, table-row, table-row-group, table-header-group, table-footer-group (which is the default for HTML tables, table rows, table bodies, table headers, and table footers, respectively), or inline-table.
 *  Block elements where overflow has a value other than visible and clip.
 *  display: flow-root.
 *  Elements with contain: layout, content, or paint.
 *  Flex items (direct children of the element with display: flex or inline-flex) if they are neither flex nor grid nor table containers themselves.
 *  Grid items (direct children of the element with display: grid or inline-grid) if they are neither flex nor grid nor table containers themselves.
 *  Multicol containers (elements where column-count or column-width isn't auto, including elements with column-count: 1).
 *  column-span: all should always create a new formatting context, even when the column-span: all element isn't contained by a multicol container (Spec change, Chrome bug).

*/
func needsRootContext(c frame.Container) bool {
	root := false
	if c.DisplayMode().Inner().Contains(css.FlowRootMode) {
		root = true
	} else if c.DisplayMode().Contains(css.InlineMode | css.InnerBlockMode) { // "inline-root"
		root = true
	}
	if c.DOMNode() != nil {
		props := c.DOMNode().ComputedStyles()
		overflow := props.GetPropertyValue("overflow-x")
		if c.DOMNode().NodeName() == "#document" {
			root = true
		} else if props.GetPropertyValue("float") != "none" {
			root = true
		} else if props.GetPropertyValue("position") == "absolute" || props.GetPropertyValue("fixed") == "absolute" {
			root = true
		} else if overflow != "visible" && overflow != "clip" {
			root = true
		} // TODO and other rules
	}
	return root
}

// ---------------------------------------------------------------------------

// NewContextFor creates a formatting context for a container.
// If c already has a context set, this context will  be returned.
func NewContextFor(c frame.Container) frame.Context {
	if c.Context() != nil {
		return c.Context()
	}
	inner := c.DisplayMode().Inner()
	isroot := needsRootContext(c)
	if inner.Contains(css.InnerInlineMode) {
		T().Debugf("providing inline context (root=%v) for [%v]", isroot, boxtree.ContainerName(c))
		return NewInlineContext(c, isroot)
	}
	if inner.Contains(css.InnerBlockMode) {
		if c.TreeNode().ChildCount() > 0 {
			T().Debugf("context: checking %d children", c.TreeNode().ChildCount())
			// If a block level container contains only inline level children,
			// its formatting context switches to inline
			modes := css.InlineMode
			children := c.TreeNode().Children(true)
			T().Debugf("context: children = %+v", children)
			for _, ch := range children {
				if childContainer, ok := ch.Payload.(frame.Container); ok {
					modes &= childContainer.DisplayMode().Outer()
				}
			}
			if modes.Contains(css.InlineMode) {
				T().Debugf("providing inline context (root=%v) for [%v]", isroot, boxtree.ContainerName(c))
				return NewInlineContext(c, isroot)
			}
		}
		T().Debugf("providing block context (root=%v) for [%v]", isroot, boxtree.ContainerName(c))
		return NewBlockContext(c, isroot)
	}
	return nil
}

func lastbox(ctx frame.Context) *frame.Box {
	children := ctx.Contained()
	if len(children) == 0 {
		return nil
	}
	return children[len(children)-1].CSSBox()
}
