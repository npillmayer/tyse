package style

import (
	"github.com/npillmayer/tyse/engine/tree"
	"golang.org/x/net/html"
)

// Styler is an interface all concrete types of styled tree nodes
// will have to implement to be usable for layout, rendering, etc.
type Styler interface {
	HTMLNode() *html.Node
	Styles() *PropertyMap
	StylesCascade() Styler
}

// Interf is a mapper from a concrete tree node to an interface
// implementation for Styler. You can think of this function type as
// an adapter from a certain tree implementation to a styled tree.
type Interf func(*tree.Node) Styler

// NodeCreator is a function to create a style node for a given HTML node.
// This interface is used to de-couple concrete implementations of styled trees
// from the building process of a styled tree. Starting with a CSS object model
// (CSSOM) an HTML parse tree has to be traversed and the CSS properites will
// be applied to each HTML node. Application of CSS styles to an HTML node will
// result in a styled tree node. NodeCreator is necessary to make this traversal
// and styled node creation independent from concrete implementations. The link
// between the styling algorithm and the concrete style tree implementation is
// a NodeCreator, which knows how to create a styled tree node from an HTML node
// and a given set of CSS styles.
type NodeCreator interface {
	StyleForHTMLNode(*html.Node) *tree.Node
	ToStyler(*tree.Node) Styler
	SetStyles(*tree.Node, *PropertyMap)
}
