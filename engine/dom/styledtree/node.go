package styledtree

import (
	"strings"

	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/tree"
	"golang.org/x/net/html"
)

// StyNode is a style node, the building block of the styled tree.
type StyNode struct {
	tree.Node[*StyNode] // we build on top of general purpose tree
	htmlNode            *html.Node
	computedStyles      *style.PropertyMap
}

func (sn *StyNode) String() string {
	h := sn.htmlNode
	switch h.Type {
	case html.DocumentNode:
		return "[#document]"
	case html.ElementNode:
		return "[" + h.Data + "]"
	case html.TextNode:
		return "[#text:" + shortText(h) + "]"
	}
	return "[styled node]"
}

// NewNodeForHTMLNode creates a new styled node linked to an HTML node.
func NewNodeForHTMLNode(html *html.Node) *tree.Node[*StyNode] {
	sn := &StyNode{}
	sn.Payload = sn // Payload will always reference the node itself
	sn.htmlNode = html
	return &sn.Node
}

// Node gets the styled node from a generic tree node.
func Node(n *tree.Node[*StyNode]) *StyNode {
	if n == nil {
		return nil
	}
	return n.Payload
}

// HTMLNode gets the HTML DOM node corresponding to this styled node.
func (sn *StyNode) HTMLNode() *html.Node {
	return sn.Payload.htmlNode
}

// StylesCascade gets the upwards to the enclosing style set.
// func (sn *StyNode) StylesCascade() *style.Styler {
// 	enclosingStyles := Node(sn.Parent())
// 	return enclosingStyles.AsStyler()
// }

// Styles is part of interface style.Styler.
func (sn *StyNode) Styles() *style.PropertyMap {
	return sn.computedStyles
}

// SetStyles sets the styling properties of a styled node.
func (sn *StyNode) SetStyles(styles *style.PropertyMap) {
	sn.computedStyles = styles
}

// GetPropertyValue returns the property value for a given key.
// If the property is inherited, it may cascade.
//func (pmap *style.PropertyMap) GetPropertyValue(key string, node *tree.Node[*styledtree.StyNode]) style.Property {
func (sn *StyNode) GetPropertyValue(key string, pmap *style.PropertyMap) style.Property {
	p, ok := pmap.Property(key)
	if ok {
		if p != "inherit" {
			return p
		}
	}
	// not found in local dicts => cascade, if allowed
	if p == "inherit" || style.IsCascading(key) {
		groupname := style.GroupNameFromPropertyKey(key)
		tracer().P("key", key).Debugf("styling: cascading for key %s", key)
		tracer().P("key", key).Debugf("styling: cascading with property group %s", groupname)
		var group *style.PropertyGroup
		for sn != nil && group == nil {
			sn = sn.Parent().Payload
		}
		if group == nil {
			return style.NullStyle
		}
		p, _ := group.Cascade(key).Get(key)
		return p
	}
	return style.NullStyle
}

// --- Helpers ---------------------------------------------------------------

// Creator returns a style-creator for use in CSSOM.
// The returned style.NodeCreator will then build up an instance of a styled tree
// with node type styledtree.StyNode.
//
/*
func Creator() style.NodeCreator {
	return creator{}
}

type creator struct{}

func (c creator) ToStyler(n *tree.Node[*StyNode]) style.Styler {
	return Node(n)
}

func (c creator) StyleForHTMLNode(htmlnode *html.Node) *tree.Node[*StyNode] {
	return NewNodeForHTMLNode(htmlnode)
}

func (c creator) SetStyles(n *tree.Node[*StyNode], m *style.PropertyMap) {
	Node(n).SetStyles(m)
}

var _ style.NodeCreator = creator{}
*/

func shortText(h *html.Node) string {
	s := "\"\\\""
	if len(h.Data) > 10 {
		s += h.Data[:10] + "...\\\"\""
	} else {
		s += h.Data + "\\\"\""
	}
	s = strings.Replace(s, "\n", `\\n`, -1)
	s = strings.Replace(s, "\t", `\\t`, -1)
	s = strings.Replace(s, " ", "\u2423", -1)
	return s
}
