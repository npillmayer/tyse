package frame

import (
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/tree"
)

type FormattingContextType uint8

// Currently we know about four formatting context types.
const (
	NoContext FormattingContextType = iota
	BlockFormattingContext
	InlineFormattingContext
	FlexFormattingContext
	GridFormattingContext
)

// Context establishes a CSS formatting context.
//
// “Boxes in the normal flow belong to a formatting context, which may be block or
// inline, but not both simultaneously. Block-level boxes participate in a block
// formatting context. Inline-level boxes participate in an inline formatting context.”
//
type Context interface {
	Type() FormattingContextType
	IsFlowRoot() bool     // this is a self-contained BFC
	Container() Container // container which creates this formatting context
}

type ContextBase struct {
	tree.Node
	container Container
	isRoot    bool
}

func (ctx ContextBase) Container() Container {
	return ctx.container
}

func (ctx ContextBase) Self() Context {
	if ctx.container == nil || ctx.container.Context() == nil {
		panic("CSS context: internal inconsistency")
	}
	return ctx.container.Context()
}

func (ctx ContextBase) TreeNode() *tree.Node {
	return &ctx.Node
}

func (ctx ContextBase) IsFlowRoot() bool {
	return ctx.isRoot
}

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
	ContextBase
}

func NewBlockContext(c Container, isRoot bool) *BlockContext {
	ctx := &BlockContext{}
	ctx.isRoot = isRoot
	ctx.container = c
	ctx.Payload = ctx
	return ctx
}

func Block(ctx Context) *BlockContext {
	if block, ok := ctx.(*BlockContext); ok {
		return block
	}
	panic("context is not a block context")
}

func (ctx *BlockContext) Type() FormattingContextType {
	return BlockFormattingContext
}

func (ctx *BlockContext) AddBox(c Container) {
	if c.DisplayMode() == InlineMode {
		anon := NewAnonymousBox(InlineMode)
		anon.AddChild(c.TreeNode())
		return
	}
	if c.DOMNode().ComputedStyles().GetPropertyValue("float") == style.NullStyle {
		T().P("context", "block").Errorf("float box cannot be added")
		panic("illegal argument for InlineContext.AddBox(c)")
	}
	if c.DOMNode().ComputedStyles().GetPropertyValue("position") == "absolute" ||
		c.DOMNode().ComputedStyles().GetPropertyValue("position") == "fixed" {
		//
		T().P("context", "block").Errorf("child container has absolute or fixed position")
		panic("illegal argument for InlineContext.AddBox(c)")
	}
	if ctx.container.TreeNode().IndexOfChild(c.TreeNode()) >= 0 {
		T().P("context", "block").Errorf("child container cannot have 2 parents")
		panic("container is child container; cannot have 2 parents")
	}
	ctx.AddChild(c.TreeNode())
}

// --- Inline Context --------------------------------------------------------

type InlineContext struct {
	ContextBase
}

func NewInlineContext(c Container, isRoot bool) *InlineContext {
	ctx := &InlineContext{}
	ctx.isRoot = isRoot
	ctx.container = c
	ctx.Payload = ctx
	return ctx
}

func Inline(ctx Context) *InlineContext {
	if inline, ok := ctx.(*InlineContext); ok {
		return inline
	}
	panic("context is not an inline context")
}

func (ctx *InlineContext) Type() FormattingContextType {
	return InlineFormattingContext
}

func (ctx *InlineContext) AddLineBox(c Container) {
	// May only add line boxes
	// inline-block boxes will already be part of the khipu
	//
	if c.DisplayMode() == InlineMode {
		ctx.AddChild(c.TreeNode())
		return
	}
	if c.DOMNode().ComputedStyles().GetPropertyValue("float") == style.NullStyle {
		T().P("context", "inline").Errorf("float box cannot be added as line box")
		panic("illegal argument for InlineContext.AddLineBox(c)")
	}
	if c.DOMNode().ComputedStyles().GetPropertyValue("position") == "absolute" ||
		c.DOMNode().ComputedStyles().GetPropertyValue("position") == "fixed" {
		//
		T().P("context", "inline").Errorf("child container has absolute or fixed position")
		panic("illegal argument for InlineContext.AddLineBox(c)")
	}
	if ctx.container.TreeNode().IndexOfChild(c.TreeNode()) >= 0 {
		T().P("context", "inline").Errorf("child container cannot have 2 parents")
		panic("container is child container; cannot have 2 parents")
	}
	anon := NewAnonymousBox(InlineMode)
	anon.AddChild(c.TreeNode())
}

// ---------------------------------------------------------------------------

/*
CreateContextForContainer is a factory method to create a suitable
formatting context for a container.

A new block formatting context is created by:

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
func CreateContextForContainer(c Container, mustRoot bool) Context {
	mode := DisplayModeForDOMNode(c.DOMNode()).BlockOrInline()
	// An inline box is one that is both inline-level and whose contents participate
	// in its containing inline formatting context. A non-replaced element with a
	// 'display' value of 'inline' generates an inline box.
	if mode == InlineMode {
		return NewInlineContext(c, mustRoot)
	}
	// c is a block level container
	// => test for special element types
	if c.DOMNode().NodeName() == "p" {
		return NewInlineContext(c, mustRoot)
	}
	block := false
	// => test for display mode of children
	props := c.DOMNode().ComputedStyles()
	overflow := props.GetPropertyValue("overflow")
	if c.DOMNode().NodeName() == "html" {
		block = true
	} else if props.GetPropertyValue("float") != "none" {
		block = true
	} else if props.GetPropertyValue("position") == "absolute" || props.GetPropertyValue("fixed") == "absolute" {
		block = true
	} else if c.DisplayMode().Contains(InlineMode | BlockMode) { // "inline-block"
		block = true
	} else if overflow != "visible" && overflow != "clip" {
		block = true
	} // TODO and other rules
	if c.TreeNode().ChildCount() > 0 {
		var modes DisplayMode
		children := c.TreeNode().Children()
		for _, ch := range children {
			if childContainer, ok := ch.Payload.(Container); ok {
				modes |= childContainer.DisplayMode().BlockOrInline()
			}
		}
		// If a block level container contains only inline level children,
		// its formatting context switches to inline
		if !modes.Contains(BlockMode) {
			mode = InlineMode
		}
	}
	if mode == InlineMode {
		return NewInlineContext(c, block || mustRoot)
	}
	return NewBlockContext(c, block || mustRoot)
}
