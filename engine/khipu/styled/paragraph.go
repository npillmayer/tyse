package styled

import (
	"fmt"
	"strings"

	"github.com/npillmayer/cords"
	sty "github.com/npillmayer/cords/styled"
	"github.com/npillmayer/tyse/engine/dom/cssom/style"
	"github.com/npillmayer/tyse/engine/dom/w3cdom"
	"golang.org/x/net/html"
)

// Paragraph represents a styled paragraph of text, from a W3C DOM.
type Paragraph struct {
	*sty.Text
}

// InnerParagraphText creates a Paragraph instance holding the text content
// of a W3C DOM node as a styled text.
// node should be a paragraph-level HTML node, but this is not enforced.
func InnerParagraphText(node w3cdom.Node) (*Paragraph, error) {
	para := &Paragraph{}
	innerText, err := innerText(node)
	if err != nil {
		return nil, err
	}
	para.Text = sty.TextFromCord(innerText)
	T().Debugf("######################################")
	//cnt := 0
	para.Text.Raw().EachLeaf(func(l cords.Leaf, pos uint64) error {
		// if cnt > 2 {
		// 	return nil
		// }
		// cnt++
		T().Debugf("creating a style leaf for '%s'", l.String())
		leaf := l.(*pLeaf)
		styles := leaf.element.ComputedStyles().Styles()
		styleset := Set{styles: styles}
		para.Text.Style(styleset, pos, pos+l.Weight())
		return nil
	})
	return para, nil
}

// ForEachStyleRun applies a function to each run of the same style set
// for a paragraph's text.
func (p *Paragraph) ForEachStyleRun(f func(run Run) error) error {
	err := p.Text.EachStyleRun(func(content string, style sty.Style, pos uint64) error {
		r := Run{
			Text:     content,
			Position: pos,
		}
		set, ok := style.(Set)
		if !ok {
			T().Errorf("paragraph each style: style is not a CSS style set")
			return cords.ErrIllegalArguments
		}
		r.StyleSet = set
		return f(r)
	})
	return err
}

// StyleAt returns the active style at text position pos, together with an
// index relative to the start of the style run.
//
// Overwrites StyleAt from cords.styled.Text
func (p *Paragraph) StyleAt(pos uint64) (Set, uint64, error) {
	sty, i, err := p.Text.StyleAt(pos)
	if err != nil {
		return Set{}, pos, err
	}
	return sty.(Set), i, nil
}

// Run is a simple container type to hold a run of text with equal style.
type Run struct {
	Text     string
	Position uint64
	StyleSet Set
}

// innerText creates a text cord for the textual content of an HTML element and all
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
func innerText(n w3cdom.Node) (cords.Cord, error) {
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
		leaf := &pLeaf{
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

// pLeaf is the leaf type for cords from a paragraph of text.
// Not intended for client usage.
type pLeaf struct {
	element w3cdom.Node
	length  uint64
	content string
}

// Weight is part of interface cords.pLeaf.
// Not intended for client usage.
func (l pLeaf) Weight() uint64 {
	return l.length
}

// String is part of interface cords.pLeaf.
// Not intended for client usage.
func (l pLeaf) String() string {
	return l.content
}

// Split is part of interface cords.pLeaf.
// Not intended for client usage.
func (l pLeaf) Split(i uint64) (cords.Leaf, cords.Leaf) {
	left := &pLeaf{
		element: l.element,
		length:  l.length,
		content: l.content[:i],
	}
	right := &pLeaf{
		element: l.element,
		length:  l.length,
		content: l.content[i:],
	}
	return left, right
}

// Substring is part of interface cords.pLeaf.
// Not intended for client usage.
func (l pLeaf) Substring(i, j uint64) []byte {
	return []byte(l.content)[i:j]
}

var _ cords.Leaf = pLeaf{}

func (l pLeaf) dbgString() string {
	e := l.element
	estr := "?"
	if e != nil {
		estr = e.NodeName()
	}
	cont := strings.Replace(l.String(), "\n", "_", -1)
	return fmt.Sprintf("{<%s> \"%s\"}", estr, cont)
}

// --- Styles pLeaf------------------------------------------------------------

// Set is a type to hold CSS styles/properties for runs of text of a Paragraph.
type Set struct {
	styles *style.PropertyMap
}

// String is part of interface cords.styled.Style.
func (set Set) String() string {
	return "<style>"
}

// Equals is part of interface cords.styled.Style, not intended for client usage.
func (set Set) Equals(other sty.Style) bool {
	if o, ok := other.(Set); ok {
		if o.styles == set.styles {
			return true
		}
	}
	return false
}

var _ sty.Style = Set{}
