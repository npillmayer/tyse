package inline

import (
	"fmt"
	"strings"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/dom/w3cdom"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/uax/bidi"
	"golang.org/x/net/html"
)

// Paragraph represents a styled paragraph of text, from a W3C DOM.
type Paragraph struct {
	*styled.Paragraph         // a Paragraph is a styled text
	irs               infoIRS // info about Bidi Isolating Run Sequences
}

type infoIRS struct {
	irsElems map[uint64]bidi.Direction // text position ⇒ explicit bidi dir
	pdis     map[uint64]bool           // text position ⇒ PDI
}

// InnerParagraphText creates a Paragraph instance holding the text content
// of a W3C DOM node as a styled text.
// node should be a paragraph-level HTML node, but this is not enforced.
func InnerParagraphText(node *dom.W3CNode) (*Paragraph, error) {
	para := &Paragraph{
		irs: infoIRS{
			irsElems: make(map[uint64]bidi.Direction),
			pdis:     make(map[uint64]bool),
		},
	}
	innerText, err := innerText(node, &para.irs)
	if err != nil {
		return nil, err
	}
	// var explicit bool
	// text := styled.TextFromCord(innerText)
	// text.Raw().EachLeaf(func(l cords.Leaf, pos uint64) error {
	// 	T().Debugf("styled paragraph: creating a style leaf for '%s'", l.String())
	// 	leaf := l.(*pLeaf)
	// 	styles := leaf.element.ComputedStyles().Styles()
	// 	styleset := Set{styles: styles}
	// 	styleset.eBidiDir, explicit = findEmbeddingBidiDirection(leaf.element)
	// 	text.Style(styleset, pos, pos+l.Weight())
	// 	if explicit {
	// 		para.irsElems[pos] = styleset.eBidiDir
	// 		para.pdis[pos+l.Weight()] = true
	// 	}
	// 	return nil
	// })
	eBidiDir, _ := findEmbeddingBidiDirection(node)
	para.Paragraph, err = styled.ParagraphFromText(innerText, 0, innerText.Raw().Len(), eBidiDir,
		para.bidiMarkup())
	return para, err
}

// ForEachStyleRun applies a function to each run of the same style set
// for a paragraph's text.
// func (p *Paragraph) ForEachStyleRun(f func(run Run) error) error {
// 	err := p.Text.EachStyleRun(func(content string, style sty.Style, pos uint64) error {
// 		r := Run{
// 			Text:     content,
// 			Position: pos,
// 		}
// 		set, ok := style.(Set)
// 		if !ok {
// 			T().Errorf("paragraph each style: style is not a CSS style set")
// 			return cords.ErrIllegalArguments
// 		}
// 		r.StyleSet = set
// 		return f(r)
// 	})
// 	return err
// }

// StyleAt returns the active style at text position pos, together with an
// index relative to the start of the style run.
//
// Overwrites StyleAt from cords.styled.Text
// func (p *Paragraph) StyleAt(pos uint64) (Set, uint64, error) {
// 	sty, i, err := p.Text.StyleAt(pos)
// 	if err != nil {
// 		return Set{}, pos, err
// 	}
// 	return sty.(Set), i, nil
// }

// Run is a simple container type to hold a run of text with equal style.
/*
type Run struct {
	Text     string
	Position uint64
	StyleSet Set
}

// Len is a shortcut for len(r.Text)
func (r Run) Len() uint64 {
	return uint64(len(r.Text))
}
*/

// innerText creates a text cord for the textual content of an HTML element and all
// its descendents. It resembles the text produced by
//
//      document.getElementById("myNode").innerText
//
// in JavaScript. However, it creates a styled text, which means that attributes
// of sub-elements are respected and preserved.
//
// The fragment organization of the resulting cord(s) will reflect the hierarchy of
// the element node's descendents.
//
func innerText(n w3cdom.Node, irs *infoIRS) (*styled.Text, error) {
	if n == nil {
		return styled.TextFromString(""), cords.ErrIllegalArguments
	}
	b := styled.NewTextBuilder()
	collectText(n, b, irs)
	return b.Text(), nil
}

func collectText(n w3cdom.Node, b *styled.TextBuilder, irs *infoIRS) {
	if n.NodeType() == html.ElementNode {
		if n.ComputedStyles().GetPropertyValue("display") == "block" {
			b.Append(&nonReplacableElementLeaf{n}, frame.StyleSet{})
		} else {
			T().Debugf("styled paragraph: collect text of <%s>", n.NodeName())
		}
	} else if n.NodeType() == html.TextNode {
		//T().Debugf("text = %s", n.NodeValue())
		parent := n.ParentNode()
		for parent != nil && parent.NodeType() != html.ElementNode {
			parent = parent.ParentNode()
		}
		//T().Debugf("parent of text node = %v", parent)
		value := n.NodeValue()
		leaf := &pLeaf{
			element: parent,
			length:  uint64(len(value)),
			content: value,
		}
		styles := parent.ComputedStyles().Styles()
		styleset := frame.StyleSet{Props: styles}
		var isExplicitDir bool
		styleset.EmbBidiDir, isExplicitDir = findEmbeddingBidiDirection(leaf.element)
		if isExplicitDir {
			irs.irsElems[b.Len()] = styleset.EmbBidiDir
			irs.pdis[b.Len()+leaf.Weight()] = true
		}
		//text.Style(styleset, pos, pos+l.Weight())
		b.Append(leaf, styleset)
	}
	if n.HasChildNodes() {
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			collectText(c, b, irs)
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

// findEmbeddingBidiDirection finds out style settings which determine the
// embedding text direction for this HTML node.
// Bidi directions in HTML may either be set with an attribute `dir` (highest
// priority) or with CSS property `direction`. We treat L2R as the default,
// only switching it for explicit setting of R2L.
func findEmbeddingBidiDirection(pnode w3cdom.Node) (bidi.Direction, bool) {
	eBidiDir, explicit := bidi.LeftToRight, false
	attrset := false
	if pnode.HasAttributes() {
		dirattr := pnode.Attributes().GetNamedItem("dir")
		attrset, explicit = true, true
		if dirattr.Value() == "rtl" {
			eBidiDir = bidi.RightToLeft
		}
	}
	if !attrset {
		propmap := pnode.ComputedStyles().Styles()
		var textDir style.Property
		textDir, explicit = style.GetLocalProperty(propmap, "direction")
		//textDir := pnode.ComputedStyles().GetPropertyValue("direction")
		if textDir == "rtl" {
			eBidiDir = bidi.RightToLeft
			explicit = true // TODO only set if property is local to pnode
		}
	}
	return eBidiDir, explicit
}

func (para *Paragraph) bidiMarkup() bidi.OutOfLineBidiMarkup {
	irs, pdis := para.irs.irsElems, para.irs.pdis
	return func(pos uint64) int {
		if i, ok := irs[pos]; ok {
			if i == bidi.LeftToRight {
				return int(bidi.MarkupLRI)
			}
			return int(bidi.MarkupRLI)
		}
		if pdi := pdis[pos]; pdi {
			return int(bidi.MarkupPDI)
		}
		return 0
	}
}

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

type nonReplacableElementLeaf struct {
	Node w3cdom.Node
}

// Weight is part of interface cords.pLeaf.
// Not intended for client usage.
func (l nonReplacableElementLeaf) Weight() uint64 {
	return 1
}

// String is part of interface cords.pLeaf.
// Not intended for client usage.
func (l nonReplacableElementLeaf) String() string {
	return "<" + l.Node.NodeName() + "/>"
}

// Split is part of interface cords.pLeaf.
// Not intended for client usage.
func (l *nonReplacableElementLeaf) Split(i uint64) (cords.Leaf, cords.Leaf) {
	return l, nil
}

// Substring is part of interface cords.pLeaf.
// Not intended for client usage.
func (l nonReplacableElementLeaf) Substring(i, j uint64) []byte {
	return []byte{}
}

var _ cords.Leaf = &nonReplacableElementLeaf{}
