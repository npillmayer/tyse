package frame

import (
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/tree"
)

// Context establishes a CSS formatting context.
//
// “Boxes in the normal flow belong to a formatting context, which may be block or
// inline, but not both simultaneously. Block-level boxes participate in a block
// formatting context. Inline-level boxes participate in an inline formatting context.”
//
type Context interface {
	TreeNode() *tree.Node
	//AddContainer(Container) // for block context only ?
	IsBlock() bool
}

type ContextBase struct {
	tree.Node
	Container Container
}

func (ctx ContextBase) Self() Context {
	if ctx.Container == nil || ctx.Container.Context() == nil {
		panic("CSS context: internal inconsistency")
	}
	return ctx.Container.Context()
}

func (ctx ContextBase) TreeNode() *tree.Node {
	return &ctx.Node
}

func (ctx ContextBase) IsBlock() bool {
	self := ctx.Self()
	if _, ok := self.(*BlockContext); ok {
		return true
	}
	return false
}

func (ctx *BlockContext) AddChildContext(childctx Context) {
	ctx.AddChild(childctx.TreeNode())
}

// --- Block Context ---------------------------------------------------------

// BlockContext establishes a CSS block formatting context.
//
// “Block-level boxes are boxes that participate in a block formatting context.
// Each block-level element generates a principal block-level box that contains
// descendant boxes and generated content and is also the box involved in any
// positioning scheme. Some block-level elements may generate additional boxes
// in addition to the principal box [for example,]: 'list-item' elements. These
// additional boxes are placed with respect to the principal box.”
//
type BlockContext struct {
	ContextBase
}

func NewBlockContext(c Container) *BlockContext {
	ctx := &BlockContext{}
	ctx.Container = c
	ctx.Payload = ctx
	return ctx
}

func Block(ctx Context) *BlockContext {
	if block, ok := ctx.(*BlockContext); ok {
		return block
	}
	panic("context is not a block context")
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
	if ctx.Container.TreeNode().IndexOfChild(c.TreeNode()) >= 0 {
		T().P("context", "block").Errorf("child container cannot have 2 parents")
		panic("container is child container; cannot have 2 parents")
	}
	ctx.AddChild(c.TreeNode())
}

// --- Inline Context --------------------------------------------------------

type InlineContext struct {
	ContextBase
}

func NewInlineContext(c Container) *InlineContext {
	ctx := &InlineContext{}
	ctx.Container = c
	ctx.Payload = ctx
	return ctx
}

func Inline(ctx Context) *InlineContext {
	if inline, ok := ctx.(*InlineContext); ok {
		return inline
	}
	panic("context is not an inline context")
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
	if ctx.Container.TreeNode().IndexOfChild(c.TreeNode()) >= 0 {
		T().P("context", "inline").Errorf("child container cannot have 2 parents")
		panic("container is child container; cannot have 2 parents")
	}
	anon := NewAnonymousBox(InlineMode)
	anon.AddChild(c.TreeNode())
}

// ---------------------------------------------------------------------------

// CreateContextForContainer is a factory method to create a suitable
// formatting context for a container.
func CreateContextForContainer(c Container) Context {
	mode := DisplayModeForDOMNode(c.DOMNode()).BlockOrInline()
	// An inline box is one that is both inline-level and whose contents participate
	// in its containing inline formatting context. A non-replaced element with a
	// 'display' value of 'inline' generates an inline box.
	if mode == InlineMode {
		return NewInlineContext(c)
	}
	// c is a block level container
	// => test for special element types
	if c.DOMNode().NodeName() == "p" {
		return NewInlineContext(c)
	}
	// => test for display mode of children
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
		return NewInlineContext(c)
	}
	return NewBlockContext(c)
}
