package tree

/*
License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>

*/

import (
	"fmt"
	"sync"
)

/*
We manage a tree of mutable nodes. Each nodes carries a payload of type parameter T.
Nodes maintain a slice of children.

In the future, we may move to immutable nodes to reduce lock contention, but first let's get
some experience with this one.
*/

// Node is the base type our tree is built of.
type Node[T comparable] struct {
	parent   *Node[T]         // parent node of this node
	children childrenSlice[T] // mutex-protected slice of children nodes
	Payload  T                // nodes may carry a payload of arbitrary type
	Rank     uint32           // rank is used for preserving sequence
}

// NewNode creates a new tree node with a given payload.
func NewNode[T comparable](payload T) *Node[T] {
	return &Node[T]{Payload: payload}
}

func (node *Node[T]) String() string {
	return fmt.Sprintf("(Node #ch=%d %v)", node.ChildCount(), node.Payload)
}

// AddChild inserts a new child node into the tree.
// The newly inserted node is connected to this node as its parent.
// It returns the parent node to allow for chaining.
//
// This operation is concurrency-safe.
func (node *Node[T]) AddChild(ch *Node[T]) *Node[T] {
	if ch != nil {
		node.children.addChild(ch, node)
	}
	return node
}

// SetChildAt inserts a new child node into the tree.
// The newly inserted node is connected to this node as its parent.
// The child is set at a given position in relation to other children,
// replacing the child at position i if it exists.
// It returns the parent node to allow for chaining.
//
// This operation is concurrency-safe.
func (node *Node[T]) SetChildAt(i int, ch *Node[T]) *Node[T] {
	if ch != nil {
		node.children.setChild(i, ch, node)
	}
	return node
}

// InsertChildAt inserts a new child node into the tree.
// The newly inserted node is connected to this node as its parent.
// The child is set at a given position in relation to other children,
// shifting children at later positions.
// It returns the parent node to allow for chaining.
//
// This operation is concurrency-safe.
func (node *Node[T]) InsertChildAt(i int, ch *Node[T]) *Node[T] {
	if ch != nil {
		node.children.insertChildAt(i, ch, node)
	}
	return node
}

// Parent returns the parent node or nil (for the root of the tree).
func (node *Node[T]) Parent() *Node[T] {
	return node.parent
}

// Isolate removes a node from its parent.
// Isolate returns the isolated node.
func (node *Node[T]) Isolate() *Node[T] {
	if node != nil && node.parent != nil {
		node.parent.children.remove(node)
	}
	return node
}

// ChildCount returns the number of children-nodes for a node
// (concurrency-safe).
func (node *Node[T]) ChildCount() int {
	return node.children.length()
}

// Child is a concurrency-safe way to get a children-node of a node.
func (node *Node[T]) Child(n int) (*Node[T], bool) {
	if n < 0 || node.children.length() <= n {
		return nil, false
	}
	ch := node.children.child(n)
	return ch, ch != nil
}

// Children returns a slice with all children of a node.
// If omitNilChildren is set, empty children aren't included in the slice
func (node *Node[T]) Children(omitNilChildren bool) []*Node[T] {
	return node.children.asSlice(omitNilChildren)
}

// IndexOfChild returns the index of a child within the list of children
// of its parent. ch may not be nil.
func (node *Node[T]) IndexOfChild(ch *Node[T]) int {
	if node.ChildCount() > 0 {
		children := node.Children(false)
		for i, child := range children {
			if ch == child {
				return i
			}
		}
	}
	return -1
}

// --- Slices of concurrency-safe sets of children ----------------------

type childrenSlice[T comparable] struct {
	sync.RWMutex
	slice []*Node[T]
}

func (chs *childrenSlice[T]) length() int {
	chs.RLock()
	defer chs.RUnlock()
	return len(chs.slice)
}

func (chs *childrenSlice[T]) addChild(child *Node[T], parent *Node[T]) {
	if child == nil {
		return
	}
	chs.Lock()
	defer chs.Unlock()
	chs.slice = append(chs.slice, child)
	child.parent = parent
}

func (chs *childrenSlice[T]) setChild(i int, child *Node[T], parent *Node[T]) {
	if child == nil {
		return
	}
	chs.Lock()
	defer chs.Unlock()
	if len(chs.slice) <= i {
		l := len(chs.slice)
		chs.slice = append(chs.slice, make([]*Node[T], i-l+1)...)
	}
	chs.slice[i] = child
	child.parent = parent
}

func (chs *childrenSlice[T]) insertChildAt(i int, child *Node[T], parent *Node[T]) {
	if child == nil {
		return
	}
	chs.Lock()
	defer chs.Unlock()
	if len(chs.slice) <= i {
		l := len(chs.slice)
		chs.slice = append(chs.slice, make([]*Node[T], i-l+1)...)
	} else {
		chs.slice = append(chs.slice, nil)   // make room for one child
		copy(chs.slice[i+1:], chs.slice[i:]) // shift i+1..n
	}
	chs.slice[i] = child
	child.parent = parent
}

func (chs *childrenSlice[T]) remove(node *Node[T]) {
	chs.Lock()
	defer chs.Unlock()
	for i, ch := range chs.slice {
		if ch == node {
			chs.slice[i] = nil
			node.parent = nil
			break
		}
	}
}

func (chs *childrenSlice[T]) child(n int) *Node[T] {
	if chs.length() == 0 || n < 0 || n >= chs.length() {
		return nil
	}
	chs.RLock()
	defer chs.RUnlock()
	return chs.slice[n]
}

func (chs *childrenSlice[T]) asSlice(omitNilCh bool) []*Node[T] {
	chs.RLock()
	defer chs.RUnlock()
	children := make([]*Node[T], 0, chs.length())
	for _, ch := range chs.slice {
		if ch != nil || !omitNilCh {
			children = append(children, ch)
		}
	}
	return children
}
