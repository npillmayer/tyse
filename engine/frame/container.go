package frame

import (
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/tree"
)

// --- Container -----------------------------------------------------------------------

// Container is an interface type for render tree nodes, i.e., boxes.
type Container interface {
	Type() ContainerType
	DOMNode() *dom.W3CNode
	TreeNode() *tree.Node
	CSSBox() *Box
	DisplayMode() css.DisplayMode // CSS display property
	Context() Context             // return the containers formatting context
	SetContext(Context)           // context will be injected
	PresetContained() bool        // pre-set contraints for children containers
	ChildIndex() int
}

type ContainerType uint8

const (
	TypeUnknown ContainerType = iota
)

// --- Base Box type ---------------------------------------------------------

type ContainerBase struct {
	tree.Node                 // a container is a node within the layout tree
	ChildInx  uint32          // this box represents child #childInx of the parent principal box
	Display   css.DisplayMode // inner and outer display mode
}

// TreeNode returns the underlying tree node for a box.
func (b *ContainerBase) TreeNode() *tree.Node {
	return &b.Node
}

// DisplayMode returns the computed display mode of this box.
func (b *ContainerBase) DisplayMode() css.DisplayMode {
	return b.Display
}

// ChildIndex returns the index of this container within the children of the enclosing container.
func (b *ContainerBase) ChildIndex() int {
	return int(b.ChildInx)
}

// Self points to the implementing type
func (b *ContainerBase) Self() interface{} {
	return b.Node.Payload
}
