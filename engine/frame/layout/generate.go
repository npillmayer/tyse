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
		T().Errorf("generate ACTION")
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
// width := css.DimenOption(c.DOMNode().ComputedStyles().GetPropertyValue("width"))

func attributeBoxes(boxRoot *boxtree.PrincipalBox) error {
	if boxRoot == nil {
		return nil
	}
	walker := tree.NewWalker(boxRoot.TreeNode())
	future := walker.TopDown(makeAttributeAction(boxRoot)).Promise()
	_, err := future()
	return err
}

// Tree action: attribute each box from CSS styles.
func makeAttributeAction(root boxtree.Container) tree.Action {
	view := viewFromBoxRoot(root)
	//return func attributeFromCSS(node *tree.Node, unused *tree.Node, chpos int) (match *tree.Node, err error) {
	return func(node *tree.Node, unused *tree.Node, chpos int) (match *tree.Node, err error) {
		c := node.Payload.(boxtree.Container)
		if c == nil {
			return
		}
		T().Debugf("attributing container %+v", c.DOMNode().NodeName())
		//
		style := c.DOMNode().ComputedStyles().GetPropertyValue // function shortcut
		//
		font := "style.font()" // TODO
		//
		// TODO min-/max-w + h
		pt := css.DimenOption(style("padding-top"))
		c.CSSBox().Padding[frame.Top] = scale(pt, view, frame.Top, font)
		pr := css.DimenOption(style("padding-right"))
		c.CSSBox().Padding[frame.Right] = scale(pr, view, frame.Right, font)
		pb := css.DimenOption(style("padding-bottom"))
		c.CSSBox().Padding[frame.Bottom] = scale(pb, view, frame.Bottom, font)
		pl := css.DimenOption(style("padding-left"))
		c.CSSBox().Padding[frame.Left] = scale(pl, view, frame.Left, font)
		// TODO borders...
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
		//
		//pos := css.PositionOption(c.DOMNode()) // later during re-ordering
		//
		return
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
		switch dir {
		case frame.Top:
			d = d.ScaleFromViewport(view.size.Y)
		case frame.Right:
			d = d.ScaleFromViewport(view.size.X)
		case frame.Bottom:
			d = d.ScaleFromViewport(view.size.Y)
		case frame.Left:
			d = d.ScaleFromViewport(view.size.X)
		}
	}
	return d
}
