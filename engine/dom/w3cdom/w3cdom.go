/*
Package w3cdom defines an interface type for W3C Document Object Models.

See also https://www.w3schools.com/XML/dom_intro.asp

Status

Early draft—API may change frequently. Please stay patient.

___________________________________________________________________________

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>

*/
package w3cdom

import (
	"github.com/npillmayer/tyse/engine/dom/style"
	"golang.org/x/net/html"
)

// Node represents W3C-type Node
type Node interface {
	NodeType() html.NodeType        // type of the underlying HTML node (ElementNode, TextNode, etc.)
	NodeName() string               // node name output depends on the node's type
	NodeValue() string              // node value output depends on the node's type
	HasAttributes() bool            // check for existence of attributes
	ParentNode() Node               // get the parent node, if any
	HasChildNodes() bool            // check for existende of sub-nodes
	ChildNodes() NodeList           // get a list of all children-nodes
	Children() NodeList             // get a list of element child-nodes
	FirstChild() Node               // get the first children-node
	NextSibling() Node              // get the Node's next sibling or nil if last
	Attributes() NamedNodeMap       // get all attributes of a node
	ComputedStyles() ComputedStyles // get computed CSS styles
	TextContent() (string, error)   // get text from node and all descendents
}

// NodeList represents W3C-type NodeList
type NodeList interface {
	Length() int
	Item(int) Node
	String() string
}

// Attr represents W3C-type Attr
type Attr interface {
	Namespace() string
	Key() string
	Value() string
}

// NamedNodeMap represents w3C-type NamedNodeMap
type NamedNodeMap interface {
	Length() int
	Item(int) Attr
	GetNamedItem(string) Attr
}

// ComputedStyles represents a CSS style
type ComputedStyles interface {
	GetPropertyValue(string) style.Property
	Styles() *style.PropertyMap
}
