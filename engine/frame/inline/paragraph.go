package inline

import (
	"fmt"
	"strings"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/cords/styled"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/dom/w3cdom"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
	"github.com/npillmayer/tyse/engine/frame/khipu"
	"github.com/npillmayer/tyse/engine/tree"
	"github.com/npillmayer/uax/bidi"
	"golang.org/x/net/html"
)

type ParagraphBox struct {
	frame.ContainerBase
	Box     *frame.StyledBox // styled box for a DOM node
	domNode *dom.W3CNode     // the DOM node this PrincipalBox refers to
	context frame.Context    // principal boxes may establish a context
	para    *Paragraph       // text of the paragraph
	khipu   *khipu.Khipu     // khipu knots make from paragraph text
	//tree.Node                     // a paragraph box is a node within the layout tree
	//displayMode frame.DisplayMode // outer display mode
}

var _ frame.Container = &ParagraphBox{}

// TreeNode returns the underlying tree node for a box.
func (pbox *ParagraphBox) TreeNode() *tree.Node {
	return &pbox.Node
}

// DOMNode returns the underlying DOM node for a render tree element.
func (pbox *ParagraphBox) DOMNode() *dom.W3CNode {
	return pbox.domNode
}

// CSSBox returns the underlying box of a render tree element.
func (pbox *ParagraphBox) CSSBox() *frame.Box {
	return &pbox.Box.Box
}

// Type returns TypeParagraph
func (pbox *ParagraphBox) Type() frame.ContainerType {
	return TypeParagraph
}

// IsAnonymous will always return false for a paragraph container.
// func (pbox *ParagraphBox) IsAnonymous() bool {
// 	return false
// }

// DisplayMode returns the computed display mode of this box.
// func (pbox *ParagraphBox) DisplayMode() frame.DisplayMode {
// 	return pbox.displayMode
// }

// IsText will always return false for a paragraph box.
// func (pbox *ParagraphBox) IsText() bool {
// 	return false
// }

func (pbox *ParagraphBox) Context() frame.Context {
	if pbox.context == nil {
		pbox.context = boxtree.CreateContextForContainer(pbox, false)
	}
	return pbox.context
}

func (pbox *ParagraphBox) PresetContained() bool {
	panic("TODO")
	return false
}

func (pbox *ParagraphBox) ChildIndices() (uint32, uint32) {
	return 0, 0
}

// ---------------------------------------------------------------------------

// Paragraph represents a styled paragraph of text, from a W3C DOM.
type Paragraph struct {
	*styled.Paragraph         // a Paragraph is a styled text
	irs               infoIRS // info about Bidi Isolating Run Sequences
}

type infoIRS struct {
	irsElems map[uint64]bidi.Direction // text position ⇒ explicit bidi dir
	pdis     map[uint64]bool           // text position ⇒ PDI
}

// --- Paragraph from Container Box ------------------------------------------

// ParagraphFromBox creates a Paragraph instance holding the text content
// of a container box node as a styled text.
// c should be a paragraph-level container, but this is not enforced.
//
// Returns a paragraphs's text, a list of block level containers which are
// children of c, or possibly an error.
//
func paragraphTextFromBox(c frame.Container) (*Paragraph, []frame.Container, error) {
	para := &Paragraph{
		irs: infoIRS{
			irsElems: make(map[uint64]bidi.Direction),
			pdis:     make(map[uint64]bool),
		},
	}
	var innerText *styled.Text // TODO set boxText()
	innerText, blocks, err := boxText(c, &para.irs)
	eBidiDir, _ := findEmbeddingBidiDirection(c.DOMNode())
	para.Paragraph, err = styled.ParagraphFromText(innerText, 0, innerText.Raw().Len(), eBidiDir,
		para.bidiMarkup())
	return para, blocks, err
}

func boxText(c frame.Container, irs *infoIRS) (*styled.Text, []frame.Container, error) {
	if c == nil {
		return styled.TextFromString(""), []frame.Container{}, cords.ErrIllegalArguments
	}
	b := styled.NewTextBuilder()
	var blocks []frame.Container
	collectBoxText(c, b, irs, blocks)
	return b.Text(), blocks, nil
}

func collectBoxText(c frame.Container, b *styled.TextBuilder, irs *infoIRS,
	blocks []frame.Container) {
	//
	if c.DOMNode() != nil && c.DOMNode().NodeType() == html.TextNode {
		leaf := createLeaf(c.DOMNode())
		styles := leaf.element.ComputedStyles().Styles()
		styleset := frame.StyleSet{Props: styles}
		var isExplicitDir bool
		styleset.EmbBidiDir, isExplicitDir = findEmbeddingBidiDirection(leaf.element)
		if isExplicitDir {
			irs.irsElems[b.Len()] = styleset.EmbBidiDir
			irs.pdis[b.Len()+leaf.Weight()] = true
		}
		//text.Style(styleset, pos, pos+l.Weight())
		b.Append(leaf, styleset)
	} else if c.DisplayMode().BlockOrInline() == frame.BlockMode {
		b.Append(&nonReplacableElementLeaf{c.DOMNode()}, frame.StyleSet{})
	} else {
		T().Debugf("styled paragraph: collect text of <%s>", c.DOMNode().NodeName())
	}
	if c.TreeNode().ChildCount() > 0 {
		children := c.TreeNode().Children(true)
		for _, ch := range children {
			if childContainer, ok := ch.Payload.(frame.Container); ok {
				collectBoxText(childContainer, b, irs, blocks)
			}
		}
	}
}

func createLeaf(n w3cdom.Node) *pLeaf {
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
	return leaf
}

// --- Paragraph from W3C Node -----------------------------------------------

// InnerParagraphText creates a Paragraph instance holding the text content
// of a W3C DOM node as a styled text.
// node should be a paragraph-level HTML node, but this is not enforced.
func InnerParagraphText(node w3cdom.Node) (*Paragraph, error) {
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
		leaf := createLeaf(n)
		/*
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
		*/
		//
		parent := leaf.element
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
func findEmbeddingBidiDirection(pnode w3cdom.Node) (eBidiDir bidi.Direction, explicit bool) {
	attrset := false
	if pnode.HasAttributes() {
		dirattr := pnode.Attributes().GetNamedItem("dir")
		attrset, explicit = true, true
		if dirattr.Value() == "rtl" {
			eBidiDir = bidi.RightToLeft
		}
	}
	if !attrset {
		var textDir style.Property
		textDir = css.GetLocalProperty(pnode.ComputedStyles().Styles(), "direction")
		if textDir == "rtl" {
			eBidiDir = bidi.RightToLeft
			explicit = true
		}
		textDir = pnode.ComputedStyles().GetPropertyValue("direction")
		if textDir == "rtl" {
			eBidiDir = bidi.RightToLeft
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
