package frame

import (
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/tree"
)

type FormattingContextType uint8

const (
	NoContext FormattingContextType = iota
)

// Context establishes a CSS formatting context.
//
// “Boxes in the normal flow belong to a formatting context, which may be block or
// inline, but not both simultaneously. Block-level boxes participate in a block
// formatting context. Inline-level boxes participate in an inline formatting context.”
//
type Context interface {
	Type() FormattingContextType
	Container() Container                    // container which creates this formatting context
	Contained() []Container                  // contained children
	AddContained(Container)                  // add a child to contain
	Layout(*FlowRoot) error                  // layout sub-container
	Measure() (Size, css.DimenT, css.DimenT) // return dimensions of context bounding box
	IsFlowRoot() bool                        // this is a self-contained BFC
	FlowRoot() *FlowRoot                     // non-nil if this context is a flow root
}

type FlowRoot struct {
	PositionedFloats   *FloatList
	UnpositionedFloats *FloatList
}

type ContextBase struct {
	tree.Node
	C         Container
	IsRootCtx bool
	flowRoot  *FlowRoot
}

func (ctx ContextBase) Container() Container {
	return ctx.C
}

// func (ctx ContextBase) Self() Context {
// 	if ctx.container == nil || ctx.container.Context() == nil {
// 		panic("CSS context: internal inconsistency")
// 	}
// 	return ctx.container.Context()
// }

func (ctx ContextBase) TreeNode() *tree.Node {
	return &ctx.Node
}

func (ctx ContextBase) IsFlowRoot() bool {
	return ctx.IsRootCtx
}

func (ctx ContextBase) FlowRoot() *FlowRoot {
	return ctx.flowRoot
}

func (ctx ContextBase) Contained() []Container {
	c := make([]Container, 0, ctx.TreeNode().ChildCount())
	for _, node := range ctx.TreeNode().Children(true) {
		c = append(c, node.Payload.(Container))
	}
	return c
}
