package frame

import (
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/tree"
)

// --- Container -----------------------------------------------------------------------

// ContainerInterf is an interface type for render tree nodes, i.e., boxes.
// type ContainerInterf interface {
// 	Type() ContainerType
// 	DOMNode() *dom.W3CNode
// 	TreeNode() *tree.Node
// 	CSSBox() *Box
// 	DisplayMode() css.DisplayMode // CSS display property
// 	// Context() Context             // return the containers formatting context
// 	// SetContext(Context)           // context will be injected
// 	PresetContained() bool // pre-set contraints for children containers
// 	// ChildIndex() int
// }

// type ContainerType uint8

// const (
// 	TypeUnknown ContainerType = iota
// )

// RenderTreeNode represents a node of the render tree, i.e. boxes.
type RenderTreeNode interface {
	DOMNode() *dom.W3CNode // boxes link back to nodes in the DOM.
	CSSBox() *Box          // CSS box which is the visual representation of this node
	PresetContained() bool // pre-set contraints for children containers
}

// --- Container type --------------------------------------------------------

// Container is a type for layout of the render tree.
type Container struct {
	tree.Node                  // a container is a node within the layout tree
	Display    css.DisplayMode // computed inner and outer display mode
	Context    Context         // containers may establish a context
	renderNode RenderTreeNode
	//ChildInx   uint32          // this box represents child #childInx of the parent principal box
}

func MakeContainer(renderNode RenderTreeNode) Container {
	return Container{renderNode: renderNode}
}

// TreeNode returns the underlying tree node for a box.
func (b *Container) TreeNode() *tree.Node {
	if b == nil {
		return nil
	}
	return &b.Node
}

// RenderNode returns the underlying tree node for a box.
func (b *Container) RenderNode() RenderTreeNode {
	if b == nil {
		return nil
	}
	return b.renderNode
}

func (b *Container) DOMNode() *dom.W3CNode {
	if b == nil || b.renderNode == nil {
		return nil
	}
	return b.renderNode.DOMNode()
}

func (b *Container) CSSBox() *Box {
	if b == nil || b.renderNode == nil {
		return nil
	}
	return b.renderNode.CSSBox()
}

// DisplayMode returns the computed display mode of this box.
// func (b *ContainerBase) DisplayMode() css.DisplayMode {
// 	return b.Display
// }

// ChildIndex returns the index of this container within the children of the enclosing container.
// func (b *ContainerBase) ChildIndex() int {
// 	return int(b.ChildInx)
// }

// Self points to the implementing type
// func (b *ContainerBase) Self() interface{} {
// 	return b.Node.Payload
// }
