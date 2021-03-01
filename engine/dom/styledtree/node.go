package styledtree

/*
BSD License

Copyright (c) 2017–21, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of this software nor the names of its contributors
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

// https://github.com/antchfx/xpath       XPath for parser
// https://github.com/antchfx/htmlquery   HTML DOM XPath
// https://github.com/ChrisTrenkamp/goxpath/tree/master/tree
// https://github.com/santhosh-tekuri/xpathparser  XPath parser
//
// https://github.com/beevik/etree        XPath for XML ("easy tree"), does this :-( :
// type Token interface {
//    Parent() *Element
//    // contains filtered or unexported methods
//}
//
// https://github.com/mmbros/treepath     (kind-of-)XPath for tree interface, BROKEN !
//
// https://godoc.org/github.com/jiangmitiao/ebook-go

import (
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/tree"
	"golang.org/x/net/html"
)

// StyNode is a style node, the building block of the styled tree.
type StyNode struct {
	tree.Node      // we build on top of general purpose tree
	htmlNode       *html.Node
	computedStyles *style.PropertyMap
}

var _ style.Styler = &StyNode{}

// NewNodeForHTMLNode creates a new styled node linked to an HTML node.
func NewNodeForHTMLNode(html *html.Node) *tree.Node {
	sn := &StyNode{}
	sn.Payload = sn // Payload will always reference the node itself
	sn.htmlNode = html
	return &sn.Node
}

// Node gets the styled node from a generic tree node.
func Node(n *tree.Node) *StyNode {
	if n == nil {
		return nil
	}
	sn, ok := n.Payload.(*StyNode)
	if ok {
		return sn
	}
	return nil
}

// AsStyler returns a styled tree node as 'style.Styler'.
func (sn *StyNode) AsStyler() style.Styler {
	return sn
}

// HTMLNode gets the HTML DOM node corresponding to this styled node.
func (sn *StyNode) HTMLNode() *html.Node {
	return sn.Payload.(*StyNode).htmlNode
}

// StylesCascade gets the upwards to the enclosing style set.
func (sn *StyNode) StylesCascade() style.Styler {
	enclosingStyles := Node(sn.Parent())
	if enclosingStyles == nil {
		T().Errorf("styled tree: enclosing style set is null! user-agent styles unset?")
	}
	return enclosingStyles.AsStyler()
}

// Styles is part of interface style.Styler.
func (sn *StyNode) Styles() *style.PropertyMap {
	return sn.computedStyles
}

// SetStyles sets the styling properties of a styled node.
func (sn *StyNode) SetStyles(styles *style.PropertyMap) {
	sn.computedStyles = styles
}

// --- styled-node creator ---------------------------------------------------

// Creator returns a style-creator for use in CSSOM.
// The returned style.NodeCreator will then build up an instance of a styled tree
// with node type styledtree.StyNode.
//
func Creator() style.NodeCreator {
	return creator{}
}

type creator struct{}

func (c creator) ToStyler(n *tree.Node) style.Styler {
	return Node(n)
}

func (c creator) StyleForHTMLNode(htmlnode *html.Node) *tree.Node {
	return NewNodeForHTMLNode(htmlnode)
}

func (c creator) SetStyles(n *tree.Node, m *style.PropertyMap) {
	Node(n).SetStyles(m)
}

var _ style.NodeCreator = creator{}
