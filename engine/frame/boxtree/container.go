package boxtree

import (
	"fmt"
	"strings"

	"github.com/npillmayer/tyse/core/option"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/tree"
)

var _ frame.RenderTreeNode = &PrincipalBox{}
var _ frame.RenderTreeNode = &AnonymousBox{}
var _ frame.RenderTreeNode = &TextBox{}

// const (
// 	TypePrincipal frame.ContainerType = 100
// 	TypeAnonymous frame.ContainerType = 101
// 	TypeText      frame.ContainerType = 102
// )

// IsPrincipal returns true if this is a principal box.
//
// Some HTML elements create a mini-hierachy of boxes for rendering. The outermost box
// is called the principal box. It will always refer to the styled node.
// An example would be a `<li>`-element: it will create two sub-boxes, one for the
// list item marker and one for the item's text/content. Another example are anonymous
// boxes, which will be generated for reconciling context/level-discrepancies.
//
func IsPrincipal(c frame.RenderTreeNode) bool {
	_, ok := c.(*PrincipalBox)
	return ok
}

// IsAnonymous returns true if this box is an anonymous box created by the layout algorithm.
func IsAnonymous(c frame.RenderTreeNode) bool {
	_, ok := c.(*AnonymousBox)
	return ok
}

// IsText returns true if the underlying box is a text box.
// Text boxes reference text nodes in the DOM.
func IsText(c frame.RenderTreeNode) bool {
	_, ok := c.(*TextBox)
	return ok
}

// --- PrincipalBox --------------------------------------------------------------------

// PrincipalBox is a (CSS-)styled box which may contain other boxes.
// It references a node in the styled tree, i.e., a stylable DOM element node.
type PrincipalBox struct {
	frame.Container
	Box     *frame.StyledBox // styled box for a DOM node
	domNode *dom.W3CNode     // the DOM node this PrincipalBox refers to
}

// NewPrincipalBox creates either a block-level container or an inline-level container
func NewPrincipalBox(domnode *dom.W3CNode, mode css.DisplayMode) *PrincipalBox {
	pbox := &PrincipalBox{
		domNode: domnode,
	}
	pbox.Display = mode
	pbox.Box = &frame.StyledBox{}
	pbox.Payload = pbox // always points to itself: tree node -> box
	return pbox
}

// TreeNodeAsPrincipalBox retrieves the payload of a tree node as a PrincipalBox.
// Will be called from clients as
//
//    box := layout.PrincipalBoxFromNode(n)
//
func TreeNodeAsPrincipalBox(n *tree.Node) *PrincipalBox {
	if n == nil {
		return nil
	}
	pbox, ok := n.Payload.(*PrincipalBox)
	if ok {
		return pbox
	}
	return nil
}

// TreeNode returns the underlying tree node for a box.
// func (pbox *PrincipalBox) TreeNode() *tree.Node {
// 	return &pbox.Node
// }

// DOMNode returns the underlying DOM node for a render tree element.
func (pbox *PrincipalBox) DOMNode() *dom.W3CNode {
	return pbox.domNode
}

// CSSBox returns the underlying box of a render tree element.
func (pbox *PrincipalBox) CSSBox() *frame.Box {
	return &pbox.Box.Box
}

// Type returns TypePrincipal
// func (pbox *PrincipalBox) Type() frame.ContainerType {
// 	return TypePrincipal
// }

// IsAnonymous will always return false for a container.
// func (pbox *PrincipalBox) IsAnonymous() bool {
// 	return false
// }

// IsText will always return false for a principal box.
// func (pbox *PrincipalBox) IsText() bool {
// 	return false
// }

// DisplayMode returns the computed display mode of this box.
// func (pbox *PrincipalBox) DisplayMode() frame.DisplayMode {
// 	//return pbox.outerMode, pbox.innerMode
// 	return pbox.displayMode
// }

// func (pbox *PrincipalBox) Context() frame.Context {
// if pbox.context == nil {
// panic("principal box does not yet have a valid context")
//pbox.context = layout.CreateContextForContainer(pbox, false)
//
// if pbox.context == nil {
// 	parent := pbox.Node.Parent()
// 	for parent != nil {
// 		c, ok := parent.Payload.(Container)
// 		if !ok {
// 			break
// 		}
// 		ctx := c.Context()
// 		if ctx != nil {
// 			pbox.context = ctx
// 		}
// 		parent = parent.Parent()
// 	}
// }
// }
//
// return pbox.context
// }

func (pbox *PrincipalBox) SetContext(ctx frame.Context) {
	pbox.Context = ctx
}

// func (pbox *PrincipalBox) String() string {
// 	if pbox == nil {
// 		return "<empty box>"
// 	}
// 	name := pbox.DOMNode().NodeName()
// 	innerSym := pbox.displayMode.Symbol()
// 	//outerSym := pbox.outerMode.Symbol()
// 	outerSym := frame.NoMode.Symbol()
// 	if pbox.context != nil {
// 		if pbox.context.Type() == BlockFormattingContext {
// 			outerSym = frame.BlockMode.Symbol()
// 		} else {
// 			outerSym = frame.InlineMode.Symbol()
// 		}
// 	}
// 	//return fmt.Sprintf("%s %s %s", outerSym, innerSym, name)
// 	return fmt.Sprintf("%s %s %s", outerSym, innerSym, name)
// }

// ChildIndices returns the positional index of this box reference to
// the parent principal box. To comply with the Container interface, it returns
// the index twice (from, to).
// func (pbox *PrincipalBox) ChildIndices() (uint32, uint32) {
// 	return pbox.ChildInx, pbox.ChildInx
// }

// func (pbox *PrincipalBox) ChildIndex() int {
//	return int(pbox.ChildInx)
// }

// func (pbox *PrincipalBox) PrepareAnonymousBoxes() {
// 	if pbox.domNode.HasChildNodes() {
// 		if pbox.displayMode.Contains(InlineMode) {
// 			// In inline mode all block-children have to be wrapped in an anon box.
// 			blockChPos := pbox.checkForChildrenWithframe.DisplayMode(BlockMode)
// 			if !blockChPos.Empty() { // yes, found
// 				// At least one block child present => need anon box for block children
// 				pbox.anonMask = blockChPos
// 				anonpos := blockChPos.Condense()
// 				for i, intv := range blockChPos {
// 					anon := NewAnonymousBox(InlineMode)
// 					anon.ChildInxFrom = intv.from
// 					anon.ChildInxTo = intv.from + intv.len - 1
// 					pbox.SetChildAt(int(anonpos[i]), anon.TreeNode())
// 				}
// 			}
// 		}
// 		if pbox.displayMode.Contains(BlockMode) {
// 			// In flow mode all children must have the same outer display mode,
// 			// either block or inline.
// 			// TODO This holds for flow and grid, too ?! others?
// 			inlineChPos := pbox.checkForChildrenWithframe.DisplayMode(InlineMode)
// 			if !(pbox.checkForChildrenWithframe.DisplayMode(BlockMode).Empty() ||
// 				inlineChPos.Empty()) { // found both
// 				// Both inline and block children => need anon boxes for inline children
// 				T().Debugf("Creating inline anon boxes at %s", inlineChPos)
// 				pbox.anonMask = inlineChPos
// 				anonpos := inlineChPos.Condense()
// 				for i, intv := range inlineChPos {
// 					anon := NewAnonymousBox(BlockMode)
// 					anon.ChildInxFrom = intv.from
// 					anon.ChildInxTo = intv.from + intv.len - 1
// 					pbox.SetChildAt(int(anonpos[i]), anon.TreeNode())
// 				}
// 			}
// 		}
// 	}
// }

// func (pbox *PrincipalBox) checkForChildrenWithframe.DisplayMode(dispMode frame.DisplayMode) runlength {
// 	domchildren := pbox.domNode.ChildNodes()
// 	var rl runlength
// 	var openintv intv
// 	for i := 0; i < domchildren.Length(); i++ {
// 		domchild := domchildren.Item(i).(*dom.W3CNode)
// 		outerMode, _ := frame.DisplayModesForDOMNode(domchild)
// 		if outerMode.Overlaps(dispMode) {
// 			if openintv != nullintv {
// 				openintv.len++
// 			} else {
// 				openintv = intv{uint32(i), uint32(1)}
// 			}
// 		} else {
// 			if openintv.len > 0 {
// 				rl = append(rl, openintv)
// 			}
// 			openintv = nullintv
// 		}
// 	}
// 	if openintv.len > 0 {
// 		rl = append(rl, openintv)
// 	}
// 	return rl
// }

// ErrNullChild flags an error condition when a non-nil child has been expected.
var ErrNullChild = fmt.Errorf("child box max not be null")

// ErrAnonBoxNotFound flags an error condition where an anonymous box should be
// present but could not be found.
var ErrAnonBoxNotFound = fmt.Errorf("no anonymous box found for index")

// AddChild appends a child box to its parent principal box.
// The child is a principal box itself, i.e. references a styleable DOM node.
// The child must have its child index set.
func (pbox *PrincipalBox) AddChild(child *PrincipalBox, at int) error {
	return pbox.addChildContainer(&child.Container, at)
}

// AddTextChild appends a child box to its parent principal box.
// The child is a text box, i.e., references a HTML text node.
// The child must have its child index set.
func (pbox *PrincipalBox) AddTextChild(child *TextBox, at int) error {
	if child == nil {
		return ErrNullChild
	}
	err := pbox.addChildContainer(&child.Container, at)
	// if err == nil {
	// 	if pbox.innerMode.Contains(InlineMode) {
	// 		child.outerMode.Set(InlineMode)
	// 	} else if pbox.innerMode.Contains(BlockMode) {
	// 		child.outerMode.Set(BlockMode)
	// 	}
	// }
	return err
}

func (pbox *PrincipalBox) addChildContainer(child *frame.Container, at int) error {
	if child == nil {
		return ErrNullChild
	}
	//child.
	// inx, _ := child.ChildIndices()
	// anon, ino, j := pbox.anonMask.Translate(inx)
	// T().Debugf("Anon mask of %s is %s, transl child #%d to %v->(%d,%d)",
	// 	pbox.String(), pbox.anonMask, inx, anon, ino, j)
	// var node *tree.Node
	// var ok bool
	// if anon {
	// 	// we will add the child to an anonymous box
	// 	node, ok = pbox.TreeNode().Child(int(ino))
	// 	if !ok { // oops, we expected an anonymous box there
	// 		return ErrAnonBoxNotFound
	// 	}
	// } else {
	node := pbox.TreeNode() // we will add the child to the principal box
	// }
	node.SetChildAt(at, child.TreeNode())
	return nil
}

// AppendChild appends a child box to a principal box.
// The child is a principal box itself, i.e. references a styleable DOM node.
// It is appended as the last child of pbox.
//
// If the child's display mode does not correspond to the context of pbox,
// an anonymous box may be inserterd.
//
func (pbox *PrincipalBox) AppendChild(child *PrincipalBox) {
	//if !pbox.displayMode.Overlaps(child.outerMode) {
	// create an anon box
	//anon := NewAnonymousBox(child.displayMode)
	//anon.TreeNode().AddChild(child.TreeNode())
	//pbox.TreeNode().AddChild(anon.TreeNode())
	//return
	//}
	pbox.TreeNode().AddChild(child.TreeNode())
}

func (pbox *PrincipalBox) PresetContained() bool {
	if pbox.Context == nil {
		tracer().Errorf("[%s] has no context yet, cannot preset children boxes",
			ContainerName(&pbox.Container))
		return false
	}
	children := pbox.TreeNode().Children(true)
	tracer().Debugf("[%v] pre-sets %d sub container(s)", pbox.domNode.NodeName(), len(children))
	hasAdded := false
	for _, ch := range children {
		switch b := ch.Payload.(type) {
		case *PrincipalBox:
			pos := css.PositionOption(b.DOMNode())
			doAdd, err := pos.Match(option.Of{
				option.None:            true,
				css.PositionFloatLeft:  false,
				css.PositionFloatRight: false,
				css.PositionAbsolute:   false,
				css.PositionFixed:      false,
				option.Some:            true,
			})
			if err != nil {
				tracer().Errorf("preset children container: %v", err)
				return false
			}
			if doAdd.(bool) {
				b.CSSBox().Max.W = pbox.CSSBox().W
				pbox.Context.AddContained(&b.Container)
				hasAdded = true
			}
		case *TextBox:
			pbox.Context.AddContained(&b.Container)
		case AnonymousBox:
			tracer().Errorf("unexpected anonymous box child")
		}
	}
	return hasAdded
}

// --- Anonymous Boxes -----------------------------------------------------------------

// AnonymousBox is a type for CSS anonymous boxes.
//
// From the spec: "If a container box (inline or block) has a block-level box inside it,
// then we force it to have only block-level boxes inside it."
//
// These block-level boxes may be anonymous boxes, wrapping line boxes. There are anonymous
// inline-level boxes, too. Both are not directly stylable by the user, but rather inherit the
// styles of their principal boxes.
type AnonymousBox struct {
	frame.Container
	Box *frame.Box // an anoymous box cannot be styled
	//context frame.Context
	//displayMode frame.DisplayMode // container lives in this mode (block or inline)
	// ChildInxFrom uint32            // this box represents children starting at #ChildInxFrom of the principal box
	// ChildInxTo   uint32            // this box represents children to #ChildInxTo
	//childInx uint32
	// outerMode    frame.DisplayMode // container lives in this mode (block or inline)
	// innerMode    frame.DisplayMode // context of children (block or inline)
}

// DOMNode returns the underlying DOM node for a render tree element.
// For anonymous boxes, it returns the DOM node corresponding to the parent container,
// which should be of type PrincipalBox.
func (anon *AnonymousBox) DOMNode() *dom.W3CNode {
	parent := TreeNodeAsPrincipalBox(anon.Parent())
	if parent == nil {
		return nil
	}
	return parent.DOMNode()
}

// TreeNode returns the underlying tree node for a box.
func (anon *AnonymousBox) TreeNode() *tree.Node {
	return &anon.Node
}

// CSSBox returns the underlying box of a render tree element.
func (anon *AnonymousBox) CSSBox() *frame.Box {
	return anon.Box
}

// IsAnonymous will always return true for an anonymous box.
// func (anon *AnonymousBox) IsAnonymous() bool {
// 	return true
// }

// IsText will always return false for an anonymous box.
// func (anon *AnonymousBox) IsText() bool {
// 	return false
// }

// DisplayMode returns the computed display mode of this box.
// func (anon *AnonymousBox) DisplayMode() frame.DisplayMode {
// 	return anon.displayMode
// }

// Type returns TypeAnonymous
// func (anon *AnonymousBox) Type() frame.ContainerType {
// 	return TypeAnonymous
// }

// func (anon *AnonymousBox) String() string {
// 	if anon == nil {
// 		return "<empty anon box>"
// 	}
// 	innerSym := anon.displayMode.Inner().Symbol()
// 	outerSym := anon.displayMode.Outer().Symbol()
// 	return fmt.Sprintf("%s %s", outerSym, innerSym)
// }

// ChildIndex returns the positional index of this anonymous box as a child of
// the principal box.
// func (anon *AnonymousBox) ChildIndex() int {
// 	return int(anon.childInx)
// }

// func (anon *AnonymousBox) Context() frame.Context {
// 	return anon.context
// }

// func (anon *AnonymousBox) SetContext(ctx frame.Context) {
// 	anon.context = ctx
// }

func NewAnonymousBox(mode css.DisplayMode) *AnonymousBox {
	anon := &AnonymousBox{}
	anon.Box = frame.InitEmptyBox(anon.CSSBox())
	anon.Display = mode
	anon.Payload = anon // always points to itself: tree node -> box
	return anon
}

func (anon *AnonymousBox) PresetContained() bool {
	if anon.Context == nil {
		tracer().Errorf("[%s] has no context yet, cannot preset children boxes",
			ContainerName(&anon.Container))
		return false
	}
	children := anon.TreeNode().Children(true)
	tracer().Debugf("[anon] pre-sets %d sub container(s)", len(children))
	hasAdded := false
	for _, ch := range children {
		switch b := ch.Payload.(type) {
		case *PrincipalBox:
			pos := css.PositionOption(b.DOMNode())
			doAdd, err := pos.Match(option.Of{
				option.None:            true,
				css.PositionFloatLeft:  false,
				css.PositionFloatRight: false,
				css.PositionAbsolute:   false,
				css.PositionFixed:      false,
				option.Some:            true,
			})
			if err != nil {
				tracer().Errorf("preset children container: %v", err)
				return false
			}
			if doAdd.(bool) {
				b.CSSBox().Max.W = anon.CSSBox().W
				anon.Context.AddContained(&b.Container)
				hasAdded = true
			}
		case *TextBox:
			anon.Context.AddContained(&b.Container)
		case AnonymousBox:
			tracer().Errorf("unexpected anonymous box child")
		}
	}
	return hasAdded
}

// --- Text Boxes ----------------------------------------------------------------------

// TextBox is a provisional type for CSS inline text boxes.
// It references a text node in the DOM.
// Text boxes will in a later stage be replaced by line boxes, which will subsume
// all text boxes under a common parent.
type TextBox struct {
	frame.Container
	//tree.Node              // a text box is a node within the layout tree
	Box        *frame.Box   // text box cannot be explicitely styled
	domNode    *dom.W3CNode // the DOM text-node this box refers to
	WSCollapse bool
	WSWrap     bool
	//outerMode frame.DisplayMode  // container lives in this mode (block or inline)
	//childInx uint32 // this box represents a text node at #ChildInx of the principal box
}

func NewTextBox(domnode *dom.W3CNode) *TextBox {
	tbox := &TextBox{domNode: domnode}
	tbox.Box = frame.InitEmptyBox(tbox.CSSBox())
	tbox.Display = css.InlineMode | css.InnerInlineMode
	tbox.Payload = tbox // always points to itself: tree node -> box
	return tbox
}

// DOMNode returns the underlying DOM node for a render tree element.
func (tbox *TextBox) DOMNode() *dom.W3CNode {
	return tbox.domNode
}

// CSSBox returns the underlying box of a render tree element.
func (tbox *TextBox) CSSBox() *frame.Box {
	return tbox.Box
}

// Type returns TypeText
// func (tbox *TextBox) Type() frame.ContainerType {
// 	return TypeText
// }

// TreeNode returns the underlying tree node for a box.
// func (tbox *TextBox) TreeNode() *tree.Node {
// 	return &tbox.Node
// }

// IsAnonymous will always return true for a text box.
// func (tbox *TextBox) IsAnonymous() bool {
// 	return true
// }

// IsText will always return true for a text box.
// func (tbox *TextBox) IsText() bool {
// 	return true
// }

// DisplayMode always returns inline.
// func (tbox *TextBox) DisplayMode() frame.DisplayMode {
// 	//return InlineMode, InlineMode
// 	return frame.InlineMode
// }

func (tbox *TextBox) Context() frame.Context {
	return nil
}

func (tbox *TextBox) SetContext(frame.Context) {
	// do nothing
}

func (tbox *TextBox) PresetContained() bool {
	return false
}

// ChildIndices returns the positional index of the text node in reference to
// the principal box. To comply with the PrincipalBox interface, it returns
// the index twice (from, to).
// func (tbox *TextBox) ChildIndex() int {
// 	return int(tbox.childInx)
// }

// ----------------------------------------------------------------------------------

func ContainerName(c *frame.Container) string {
	if IsText(c.RenderNode()) {
		return shortText(c.RenderNode().(*TextBox))
	}
	if c.DOMNode() == nil {
		return "anon-box"
	}
	return c.DOMNode().NodeName()
}

func shortText(tbox *TextBox) string {
	txt := tbox.domNode.NodeValue()
	//disp := box.C.DisplayMode()
	//sym := disp.Symbol()
	//s := fmt.Sprintf("\"%s\u2000\\\"", sym)
	s := fmt.Sprintf("\"%s\u2000\\\"", "T")
	if len(txt) > 10 {
		s += txt[:10] + "…\\\"\""
	} else {
		s += txt + "\\\"\""
	}
	s = strings.Replace(s, "\n", `\\n`, -1)
	s = strings.Replace(s, "\t", `\\t`, -1)
	s = strings.Replace(s, " ", "\u2423", -1)
	return s
}

/*
type runlength []intv // a list of intervals
type intv struct {    // run-length interval
	from, len uint32
}

var nullintv = intv{} // null-type for intervals

func (rl runlength) Empty() bool {
	return len(rl) == 0
}

// Condense returns a list of positions, where every interval of rl is counted
// as a single position. This gives positional indices for anonymous boxes
// associated with the intervals, usable as indices in the parents child-vector.
func (rl runlength) Condense() (positions []uint32) {
	if rl == nil {
		return positions
	}
	pos := uint32(0)
	next := uint32(0)
	for _, intv := range rl {
		if intv.from > pos {
			for j := pos; j < intv.from; j++ {
				next++
			}
		}
		positions = append(positions, next)
		next++
		pos = intv.from + intv.len
	}
	return positions
}

// Translate takes an input index (of a child node) and returns the real
// position. The boolean return value is true, if the input index lies within
// one of the intervals of rl, otherwise false.
func (rl runlength) Translate(inx uint32) (bool, uint32, uint32) {
	if rl == nil {
		return false, 0, inx // nothing to translate
	}
	last := uint32(0) // max input index processed + 1
	pos := uint32(0)  // next possible output index
	for _, intv := range rl {
		if inx < intv.from { // inx is left of this interval
			pos = pos + inx - last
			return false, uint32(0), pos
		}
		if inx <= intv.from+intv.len-1 { // inx is in this interval
			return true, pos + intv.from - last, inx - intv.from
		}
		// account for positions including the current interval
		pos = pos + intv.from - last + 1
		last = intv.from + intv.len
	}
	// inx is to the right of the last interval
	return false, uint32(0), pos + inx - last
}

func (rl runlength) String() string {
	var b bytes.Buffer
	b.WriteString("(")
	for _, iv := range rl {
		if iv.len == 0 {
			b.WriteString(" []")
		} else {
			b.WriteString(fmt.Sprintf(" [%d..%d]", iv.from, iv.from+iv.len-1))
		}
	}
	b.WriteString(" )")
	return b.String()
}
*/
