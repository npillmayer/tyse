package layout

// This module should have knowledge about:
// - which mini-hierarchy of boxes to create for each HTML element
// - which context the element should span for its children

import (
	"errors"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
	"github.com/npillmayer/tyse/engine/tree"
	"golang.org/x/net/html"
)

var errDOMRootIsNull = errors.New("DOM root is null")
var errDOMNodeNotSuitable = errors.New("DOM node is not suited for layout")

// BuildBoxTree creates a render box tree from a styled tree.
func BuildBoxTree(domRoot *dom.W3CNode) (boxtree.Container, error) {
	if domRoot == nil {
		return nil, errDOMRootIsNull
	}
	domWalker := domRoot.Walk()
	T().Debugf("Creating box tree")
	dom2box := newAssoc()
	createBoxForEach := prepareBoxCreator(dom2box)
	future := domWalker.TopDown(createBoxForEach).Promise() // start asynchronous traversal
	renderNodes, err := future()                            // wait for top-down traversal to finish
	if err != nil {
		return nil, err
	}
	T().Infof("Walker returned %d render nodes", len(renderNodes))
	/*
		for _, rnode := range renderNodes {
			n := TreeNodeAsPrincipalBox(rnode)
			T().Infof("  node for %s", n.domNode.NodeName())
		}
	*/
	T().Infof("dom2box dict contains %d entries", dom2box.Length())
	//T().Errorf("domRoot/2 = %s", dbgNodeString(domRoot))
	boxRoot, ok := dom2box.Get(domRoot)
	//T().Errorf("box for domRoot = %v", boxRoot)
	if !ok {
		T().Errorf("No box created for root style node")
	}
	return boxRoot, nil
}

// prepareBocCreator is an action function for concurrent tree-traversal.
func prepareBoxCreator(dict *domToBoxAssoc) tree.Action {
	dom2box := dict
	action := func(node *tree.Node, parentNode *tree.Node, chpos int) (*tree.Node, error) {
		T().Infof("generate ACTION box creator")
		domnode, err := dom.NodeFromTreeNode(node)
		if err != nil {
			T().Errorf("action 1: %s", err.Error())
			return nil, err
		}
		var parent *dom.W3CNode
		if parentNode != nil {
			parent, err = dom.NodeFromTreeNode(parentNode)
			if err != nil {
				T().Errorf("action 2: %s", err.Error())
				return nil, err
			}
		}
		return makeBoxNode(domnode, parent, chpos, dom2box)
	}
	return action
}

func makeBoxNode(domnode *dom.W3CNode, parent *dom.W3CNode, chpos int, dom2box *domToBoxAssoc) (
	*tree.Node, error) {
	//
	T().Infof("making box for %s", domnode.NodeName())
	box := NewBoxForDOMNode(domnode)
	if box == nil { // legit, e.g. for "display:none"
		T().Debugf("box is nil")
		return nil, nil // will not descend to children of domnode
	}
	T().Infof("remembering %d/%s", domnode.NodeType(), domnode.NodeName())
	dom2box.Put(domnode, box) // associate the styled tree node to this box
	if !domnode.IsDocument() {
		if parentNode := domnode.ParentNode(); parentNode != nil {
			parent := parentNode.(*dom.W3CNode)
			parentbox, found := dom2box.Get(parent)
			if found {
				T().Debugf("adding new box %s node to parent %s\n", box, parentbox)
				p := parentbox.(*boxtree.PrincipalBox)
				var err error
				switch b := box.(type) {
				case *boxtree.PrincipalBox:
					//b.ChildInx = uint32(chpos)
					err = p.AddChild(b, chpos)
				case *boxtree.TextBox:
					//b.ChildInx = uint32(chpos)
					err = p.AddTextChild(b, chpos)
				default:
					T().Errorf("Unknown box type for %v", box)
				}
				if err != nil {
					T().Errorf(err.Error())
				}
				_, ok := p.Child(0)
				if !ok {
					T().Errorf("Parent has no child!")
				}
			}
		}
	}
	//possiblyCreateMiniHierarchy(box)
	return box.TreeNode(), nil
}

// ----------------------------------------------------------------------

// NewBoxForDOMNode creates an adequately initialized box for a given DOM node.
func NewBoxForDOMNode(domnode *dom.W3CNode) boxtree.Container {
	if domnode.NodeType() == html.TextNode {
		tbox := boxtree.NewTextBox(domnode)
		// TODO find index within parent
		// and set #ChildInx
		return tbox
	}
	// document or element node
	mode := frame.DisplayModeForDOMNode(domnode)
	if mode == frame.NoMode || mode == frame.DisplayNone {
		return nil // do not produce box for illegal mode or for display = "none"
	}
	pbox := boxtree.NewPrincipalBox(domnode, mode)
	//pbox.PrepareAnonymousBoxes()
	// TODO find index within parent
	// and set #ChildInx
	return pbox
}

func possiblyCreateMiniHierarchy(pbox *boxtree.PrincipalBox) {
	//htmlnode := pbox.DOMNode().HTMLNode()
	//propertyMap := styler.ComputedStyles()
	switch pbox.DOMNode().NodeName() {
	case "li":
		//markertype, _ := style.GetCascadedProperty(c.DOMNode, "list-style-type", toStyler)
		markertype := pbox.DOMNode().ComputedStyles().GetPropertyValue("list-style-type")
		if markertype != "none" {
			//markerbox := newContainer(BlockMode, FlowMode)
			// TODO: fill box with correct marker symbol
			//pbox.Add(markerbox)
			T().Debugf("need marker for principal box")
		}
	}
}

// ---------------------------------------------------------------------------

// TODO initialize box with style properties affection box layout:
//    - padding
//    - border
//    - margin
//    - width
//    - height
//    - box-sizing
//
// This is probably a separate run after the box tree is complete
//
// TODO initialize whitespace flags for TextBoxes
// TODO initialize styling information for principal boxes
//
// width := css.DimenOption(c.DOMNode().ComputedStyles().GetPropertyValue("width"))

func attributeBoxes(boxRoot *boxtree.PrincipalBox) error {
	if boxRoot == nil {
		return nil
	}
	walker := tree.NewWalker(boxRoot.TreeNode())
	future := walker.TopDown(makeAttributesAction(boxRoot)).Promise()
	_, err := future()
	return err
}

// Tree action: attribute each box from CSS styles.
func makeAttributesAction(root boxtree.Container) tree.Action {
	T().Infof("generate ACTION attributer")
	view := viewFromBoxRoot(root)
	//return func attributeFromCSS(node *tree.Node, unused *tree.Node, chpos int) (match *tree.Node, err error) {
	return func(node *tree.Node, parentNode *tree.Node, chpos int) (*tree.Node, error) {
		c := node.Payload.(boxtree.Container)
		if c == nil {
			return nil, nil
		}
		parent := parentNode.Payload.(boxtree.Container)
		style := c.DOMNode().ComputedStyles().GetPropertyValue // function shortcut
		if c.Type() == boxtree.TypePrincipal {
			T().Debugf("attributing container %+v", c.DOMNode().NodeName())
			//
			// TODO font handling
			// https://developer.mozilla.org/en-US/docs/Web/CSS/font
			font := style("font-family") // TODO family, size, style, weight
			font = "style.font()"        // TODO
			//
			setSizingInformationForPrincipalBox(c, view, string(font))
			setVisualStylesForPrincipalBox(c)
			//pos := css.PositionOption(c.DOMNode()) // later during re-ordering
			//
		} else if c.Type() == boxtree.TypeText {
			setWhitespaceProperties(c, parent)
		}
		return node, nil
	}
}

func setSizingInformationForPrincipalBox(c boxtree.Container, view *view, font string) {
	//
	style := c.DOMNode().ComputedStyles().GetPropertyValue // function shortcut
	// Padding
	pt := css.DimenOption(style("padding-top"))
	c.CSSBox().Padding[frame.Top] = scale(pt, view, frame.Top, font)
	pr := css.DimenOption(style("padding-right"))
	c.CSSBox().Padding[frame.Right] = scale(pr, view, frame.Right, font)
	pb := css.DimenOption(style("padding-bottom"))
	c.CSSBox().Padding[frame.Bottom] = scale(pb, view, frame.Bottom, font)
	pl := css.DimenOption(style("padding-left"))
	c.CSSBox().Padding[frame.Left] = scale(pl, view, frame.Left, font)
	// Borders
	bt := css.DimenOption(style("border-top-width"))
	c.CSSBox().BorderWidth[frame.Top] = scale(bt, view, frame.Top, font)
	br := css.DimenOption(style("border-right-width"))
	c.CSSBox().BorderWidth[frame.Right] = scale(br, view, frame.Right, font)
	bb := css.DimenOption(style("border-bottom-width"))
	c.CSSBox().BorderWidth[frame.Bottom] = scale(bb, view, frame.Bottom, font)
	bl := css.DimenOption(style("border-left-width"))
	c.CSSBox().BorderWidth[frame.Left] = scale(bl, view, frame.Left, font)
	// Margins
	mt := css.DimenOption(style("margin-top"))
	c.CSSBox().Margins[frame.Top] = scale(mt, view, frame.Top, font)
	mr := css.DimenOption(style("margin-right"))
	c.CSSBox().Margins[frame.Right] = scale(mr, view, frame.Right, font)
	mb := css.DimenOption(style("margin-bottom"))
	c.CSSBox().Margins[frame.Bottom] = scale(mb, view, frame.Bottom, font)
	ml := css.DimenOption(style("margin-left"))
	c.CSSBox().Margins[frame.Left] = scale(ml, view, frame.Left, font)
	//
	borderSizing := style("box-sizing") == "border-box"
	c.CSSBox().BorderBoxSizing = borderSizing
	w := css.DimenOption(style("width"))
	w = scale(w, view, frame.Left, font)
	h := css.DimenOption(style("height"))
	h = scale(w, view, frame.Top, font)
	c.CSSBox().W = w
	c.CSSBox().H = h
	// TODO min-/max-w + h
}

func setVisualStylesForPrincipalBox(c boxtree.Container) {
	if c == nil || c.Type() != boxtree.TypePrincipal {
		return // other container types cannot be styled
	}
	pbox := c.(*boxtree.PrincipalBox).Box                  // box with styles
	style := c.DOMNode().ComputedStyles().GetPropertyValue // function shortcut
	fgcolor := style("color").Color()
	bgcolor := style("background-color").Color()
	bcolor := style("border-top-color").Color()
	if bcolor == nil && fgcolor != nil {
		bcolor = fgcolor // border-color = currentcolor as defined by CSS spec
	}
	if bcolor != nil || fgcolor != nil || bgcolor != nil {
		if pbox.Styles == nil {
			pbox.Styles = &frame.Styling{}
		}
		pbox.Styles.Border.LineColor = bcolor
		pbox.Styles.Colors.Foreground = fgcolor
	}
}

/*
                  New lines    Spaces and tabs     Text wrapping     End-of-line spaces
				  ---------------------------------------------------------------------
    normal        Collapse     Collapse            Wrap              Remove
    nowrap        Collapse     Collapse            No wrap           Remove
    pre           Preserve     Preserve            No wrap           Preserve
    pre-wrap      Preserve     Preserve            Wrap              Hang
    pre-line      Preserve     Collapse            Wrap              Remove
    break-spaces  Preserve     Preserve            Wrap              Wrap
*/
func setWhitespaceProperties(c, parent boxtree.Container) {
	if c != nil && parent != nil && c.Type() == boxtree.TypeText {
		t := c.(*boxtree.TextBox)
		ws := parent.DOMNode().ComputedStyles().GetPropertyValue("white-space")
		switch ws {
		case "nowrap":
			t.WSCollapse = true
			t.WSWrap = false
		case "pre":
			t.WSCollapse = false
			t.WSWrap = false
		case "pre-wrap", "pre-line", "break-spaces": // TODO
			t.WSCollapse = false
			t.WSWrap = true
		default: // white-space = normal
			t.WSCollapse = true
			t.WSWrap = true
		}
	}
}

type view struct {
	// TODO create this during DOM tree building
	font string // TODO TypeFace
	size dimen.Point
}

func viewFromBoxRoot(root boxtree.Container) *view {
	return &view{
		font: "view font",
		size: dimen.DINA4,
	}
}

func scale(d css.DimenT, view *view, dir int, font string) css.DimenT {
	if d.IsRelative() {
		if d.UnitString() == "rem" {
			d = d.ScaleFromFont(view.font)
		} else {
			d = d.ScaleFromFont(font)
		}
		d = d.ScaleFromViewport(view.size.X, view.size.Y)
		// switch dir {
		// case frame.Top:
		// case frame.Right:
		// case frame.Bottom:
		// case frame.Left:
		// }
	}
	return d
}
