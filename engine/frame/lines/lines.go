package lines

import (
	"fmt"
	"strings"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/tyse/engine/dom/w3cdom"
	"golang.org/x/net/html"
)

// InnerText creates a text cord for the textual content of an HTML element and all
// its descendents. It resembles the text produced by
//
//      document.getElementById("myNode").innerText
//
// in JavaScript (except that html.InnerText cannot respect CSS styling suppressing
// the visibility of the node's descendents).
//
// The fragment organization of the resulting cord will reflect the hierarchy of
// the element node's descendents.
//
func InnerText(n w3cdom.Node) (cords.Cord, error) {
	if n == nil {
		return cords.Cord{}, cords.ErrIllegalArguments
	}
	b := cords.NewBuilder()
	collectText(n, b)
	return b.Cord(), nil
}

func collectText(n w3cdom.Node, b *cords.CordBuilder) {
	if n.NodeType() == html.ElementNode {
		T().Debugf("<%s>", n.NodeValue)
	} else if n.NodeType() == html.TextNode {
		T().Debugf("text = %s", n.NodeValue())
		parent := n.ParentNode()
		for parent != nil && parent.NodeType() != html.ElementNode {
			parent = parent.ParentNode()
		}
		T().Debugf("parent of text node = %v", parent)
		value := n.NodeValue()
		leaf := &Leaf{
			element: parent,
			length:  uint64(len(value)),
			content: value,
		}
		b.Append(leaf)
	}
	if n.HasChildNodes() {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			collectText(c, b)
		}
	}
}

// https://www.w3schools.com/cssref/pr_text_white-space.asp
//
// TODO soll whitespace entsprechend dem white-space property behandelt
// werden und entsprechend im cord liegen ?
// oder soll dies in späteren schritten passieren? Wie wird die Position
// zugeordnet; z.B. im Vergleich zu bidi ?
// Ich denke, der Rohtext muss erst mal erhalten bleiben und das WS-collapsing
// findet während der Knotenknüpfung statt.

// ---------------------------------------------------------------------------

// Leaf is the leaf type created for cords from calls to html.InnerText(…).
type Leaf struct {
	element w3cdom.Node
	length  uint64
	content string
}

// Weight of a leaf is its string length in bytes.
func (l Leaf) Weight() uint64 {
	return l.length
}

func (l Leaf) String() string {
	return l.content
}

// Split splits a leaf at position i, resulting in 2 new leafs.
func (l Leaf) Split(i uint64) (cords.Leaf, cords.Leaf) {
	left := &Leaf{
		element: l.element,
		length:  l.length,
		content: l.content[:i],
	}
	right := &Leaf{
		element: l.element,
		length:  l.length,
		content: l.content[i:],
	}
	return left, right
}

// Substring returns a string segment of the leaf's text fragment.
func (l Leaf) Substring(i, j uint64) []byte {
	return []byte(l.content)[i:j]
}

var _ cords.Leaf = Leaf{}

func (l Leaf) dbgString() string {
	e := l.element
	estr := "?"
	if e != nil {
		estr = e.NodeName()
	}
	cont := strings.Replace(l.String(), "\n", "_", -1)
	return fmt.Sprintf("{<%s> \"%s\"}", estr, cont)
}
