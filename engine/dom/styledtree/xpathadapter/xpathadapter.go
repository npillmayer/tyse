/*
Package xpathadapter implements an xpath.NodeNavigator.

We use this library for XPath queries:

	github.com/antchfx/xpath

Package xpathadapter implements an adapter to enable antchfx/xpath to
access a styled tree, where nodes are of type styledtree.Node (the latter
is the default implementation for a styled tree). For a possible usage
of this package, refer to package dom (type dom.XPath).

For a description of the various methods of interface xpath.NodeNavigator
please refer to the documentation of antchfx/xpath. It is not replicated here.

BSD License

Copyright (c) 2017–18, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of Norbert Pillmayer nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package xpathadapter

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/antchfx/xpath"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/engine/dom/styledtree"
	"github.com/npillmayer/tyse/engine/tree"
	"golang.org/x/net/html"
)

// T traces to a global core tracer.
func T() tracing.Trace {
	return gtrace.CoreTracer
}

type NodeNavigator struct {
	root, current *styledtree.StyNode
	chinx         int // index into children slice
	attr          int // attributes index
}

// NewNavigator creates a new xpath.NodeNavigator for a styled tree.
func NewNavigator(node *styledtree.StyNode) *NodeNavigator {
	return &NodeNavigator{
		current: node,
		root:    node,
		attr:    -1,
	}
}

// CurrentNode implements dom.NodeExtractorFunc
func CurrentNode(nav xpath.NodeNavigator) (*tree.Node, error) {
	mynav, ok := nav.(*NodeNavigator)
	if !ok {
		return nil, errors.New("Navigator is not of type xpathadapter.NodeNavigator")
	}
	if mynav.current == nil {
		return nil, nil
	}
	return &mynav.current.Node, nil
}

func (nav *NodeNavigator) NodeType() xpath.NodeType {
	//T().Errorf("get node type (current = %v)", nav.current)
	switch nav.current.HTMLNode().Type {
	case html.CommentNode:
		return xpath.CommentNode
	case html.TextNode:
		return xpath.TextNode
	case html.DocumentNode:
		//T().Errorf("    node type (current = %v) is document", nav.current)
		return xpath.RootNode
	case html.ElementNode:
		if nav.attr != -1 {
			return xpath.AttributeNode
		}
		//T().Errorf("    node type (current = %v) is element", nav.current)
		return xpath.ElementNode
	case html.DoctypeNode:
		// ignored <!DOCTYPE HTML> declare and as Root-Node type.
		return xpath.RootNode
	}
	panic(fmt.Sprintf("unknown node type: %v", nav.current.HTMLNode().Type))
}

func (nav *NodeNavigator) LocalName() string {
	//T().Errorf("get local name (current = %v)", nav.current)
	if nav.attr != -1 {
		return nav.current.HTMLNode().Attr[nav.attr].Key
	}
	//T().Errorf("    local name (current = %v) is %v", nav.current, nav.current.HTMLNode().Data)
	return nav.current.HTMLNode().Data
}

func (*NodeNavigator) Prefix() string {
	return ""
}

func (nav *NodeNavigator) Value() string {
	//T().Errorf("get value (current = %v)", nav.current)
	switch nav.current.HTMLNode().Type {
	case html.CommentNode:
		return "<comment nodes not supported>"
	case html.ElementNode:
		if nav.attr != -1 {
			return nav.current.HTMLNode().Attr[nav.attr].Val
		}
		return innerText(nav.current.HTMLNode())
	case html.TextNode:
		return nav.current.HTMLNode().Data
	}
	return ""
}

func (nav *NodeNavigator) Copy() xpath.NodeNavigator {
	//T().Errorf("copy nav (current = %v)", nav.current)
	n := *nav
	return &n
}

func (nav *NodeNavigator) MoveToRoot() {
	//T().Errorf("move to root (current = %v)", nav.current)
	nav.current = nav.root
}

func (nav *NodeNavigator) MoveToParent() bool {
	//T().Errorf("move to parent (current = %v)", nav.current)
	//cname := nav.current.HTMLNode().Data
	if nav.attr != -1 {
		nav.attr = -1 // move from attributes to element
		return true
	}
	if nav.current == nav.root {
		return false
	}
	nav.current = styledtree.Node(nav.current.Parent())
	if nav.current == nil {
		return false
	}
	//T().Errorf("------ node=%v, parent=%v", cname, nav.current.HTMLNode().Data)
	nav.chinx = 0
	return true
}

func (nav *NodeNavigator) MoveToNextAttribute() bool {
	if nav.attr >= len(nav.current.HTMLNode().Attr)-1 {
		return false
	}
	nav.attr++
	return true
}

func (nav *NodeNavigator) MoveToChild() bool {
	//T().Errorf("move to child (current = %v)", nav.current)
	//cname := nav.current.HTMLNode().Data
	if nav.attr != -1 {
		return false
	}
	if nav.current.ChildCount() == 0 {
		//T().Errorf("      0 children (current = %v)", nav.current)
		return false
	}
	nav.chinx = 0
	ok := false
	child, ok := nav.current.Child(0)
	if ok {
		nav.current = styledtree.Node(child)
		//T().Errorf("move OK, child = %v", nav.current)
		//T().Errorf("------ node=%v, child=%v", cname, nav.current.HTMLNode().Data)
	}
	return ok
}

func (nav *NodeNavigator) MoveToFirst() bool {
	//T().Errorf("move to first (current = %v)", nav.current)
	if nav.attr != -1 || nav.chinx == 0 {
		return false
	}
	nav.chinx = 0
	parent := styledtree.Node(nav.current.Parent())
	child, ok := parent.Child(0)
	if ok {
		nav.current = styledtree.Node(child)
	}
	return ok
}

func (nav *NodeNavigator) String() string {
	return nav.Value()
}

func (nav *NodeNavigator) MoveToNext() bool {
	//T().Errorf("move to next (current = %v)", nav.current)
	//cname := nav.current.HTMLNode().Data
	//old := nav.current
	if nav.attr != -1 {
		return false
	}
	parent := styledtree.Node(nav.current.Parent())
	//T().Errorf("parent.ch#=%d", parent.ChildCount())
	//T().Errorf("nav.chinx=%d", nav.chinx)
	i := 0
	for ; i < parent.ChildCount()-1; i++ {
		child, ok := parent.Child(i)
		if ok {
			y := styledtree.Node(child)
			if y == nav.current {
				break
			}
		}
	}
	i++
	if i == parent.ChildCount() { // was last child of parent
		return false
	}
	child, ok := parent.Child(i)
	if ok {
		nav.current = styledtree.Node(child)
		// if old == nav.current {
		// 	for i := 0; i < parent.ChildCount(); i++ {
		// 		x, _ := parent.Child(i)
		// 		xx := styledtree.Node(x)
		// 		T().Errorf("x = %v", xx.HTMLNode().Data)
		// 	}
		// 	T().Errorf("old = %v, inx=%d", old.HTMLNode().Data, nav.chinx)
		// 	panic("old == nav.current")
		// }
	}
	return ok
}

func (nav *NodeNavigator) MoveToPrevious() bool {
	//T().Errorf("move to previous (current = %v)", nav.current)
	if nav.attr != -1 {
		return false
	}
	if nav.chinx > 0 {
		nav.chinx--
		parent := styledtree.Node(nav.current.Parent())
		ch, ok := parent.Child(nav.chinx)
		if ok {
			nav.current = styledtree.Node(ch)
		}
		return ok
	}
	return false
}

func (nav *NodeNavigator) MoveTo(other xpath.NodeNavigator) bool {
	//T().Errorf("move to other (current = %v)", nav.current)
	n, ok := other.(*NodeNavigator)
	if !ok || n.root != nav.root {
		return false
	}

	nav.current = n.current
	nav.attr = n.attr
	return true
}

var _ xpath.NodeNavigator = &NodeNavigator{}

// InnerText returns the text between the start and end tags of the object.
func innerText(n *html.Node) string {
	var output func(*bytes.Buffer, *html.Node)
	output = func(buf *bytes.Buffer, n *html.Node) {
		switch n.Type {
		case html.TextNode:
			buf.WriteString(n.Data)
			return
		case html.CommentNode:
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			output(buf, child)
		}
	}

	var buf bytes.Buffer
	output(&buf, n)
	return buf.String()
}
