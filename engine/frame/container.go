package frame

import (
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/tree"
)

// --- Container -----------------------------------------------------------------------

// ContainerInterf is an interface type for render tree nodes, i.e., boxes.
type ContainerInterf interface {
	Type() ContainerType
	DOMNode() *dom.W3CNode
	TreeNode() *tree.Node
	CSSBox() *Box
	DisplayMode() css.DisplayMode // CSS display property
	// Context() Context             // return the containers formatting context
	// SetContext(Context)           // context will be injected
	PresetContained() bool // pre-set contraints for children containers
	// ChildIndex() int
}

type ContainerType uint8

const (
	TypeUnknown ContainerType = iota
)

type RenderTreeNode interface {
	CSSBox() *Box
	DOMNode() *dom.W3CNode
	PresetContained() bool // pre-set contraints for children containers
}

// --- Base Box type ---------------------------------------------------------

type ContainerBase struct {
	tree.Node                  // a container is a node within the layout tree
	ChildInx   uint32          // this box represents child #childInx of the parent principal box
	Display    css.DisplayMode // inner and outer display mode
	Context    Context         // boxes may establish a context
	renderNode RenderTreeNode
}

// TreeNode returns the underlying tree node for a box.
func (b *ContainerBase) TreeNode() *tree.Node {
	if b == nil {
		return nil
	}
	return &b.Node
}

// RenderNode returns the underlying tree node for a box.
func (b *ContainerBase) RenderNode() RenderTreeNode {
	if b == nil {
		return nil
	}
	return b.renderNode
}

// DisplayMode returns the computed display mode of this box.
// func (b *ContainerBase) DisplayMode() css.DisplayMode {
// 	return b.Display
// }

func (b *ContainerBase) DOMNode() *dom.W3CNode {
	if b == nil || b.renderNode == nil {
		return nil
	}
	return b.renderNode.DOMNode()
}

func (b *ContainerBase) CSSBox() *Box {
	if b == nil || b.renderNode == nil {
		return nil
	}
	return b.renderNode.CSSBox()
}

// ChildIndex returns the index of this container within the children of the enclosing container.
// func (b *ContainerBase) ChildIndex() int {
// 	return int(b.ChildInx)
// }

// Self points to the implementing type
// func (b *ContainerBase) Self() interface{} {
// 	return b.Node.Payload
// }
