package inline

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/npillmayer/cords"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/domdbg"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
	"golang.org/x/net/html"
)

var dot bool = true

var testhtml = `
	<!DOCTYPE html>
	<html>
	<body>
	<h1>My First Heading</h1>
	<p>My <b>first</b> paragraph.</p>
	</body>
	</html>
`

func TestParaNode(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.frame")
	defer teardown()
	//
	domroot := buildTestDOM(`<p>My <b>bold</b> paragraph.</p>`, t)
	boxes, err := boxtree.BuildBoxTree(domroot)
	checkBoxTree(boxes, err, t)
	found, pbox := findParaContainer(boxes, t)
	if !found || pbox == nil {
		t.Fatal("no paragraph found in input text")
	}
	t.Logf("p node is of type %T / %#v", pbox, pbox)
	recursiveContextAndPreset(pbox, t)
	// now okay to layout
	if dot {
		domdbg.Dotty(pbox.DOMNode(), t)
	}
	para, _, err := paragraphTextFromBox(pbox)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if para.Raw().IsVoid() {
		t.Fatalf("inner text of para is void, should not be")
	}
	t.Logf("inner text = (%s)", para.Raw().String())
	if dot {
		cordsdotty(para.Raw(), t)
	}
}

func TestParaCreate(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.frame")
	defer teardown()
	//
	domroot := buildTestDOM(testhtml, t)
	boxes, err := boxtree.BuildBoxTree(domroot)
	checkBoxTree(boxes, err, t)
	found, pbox := findParaContainer(boxes, t)
	if !found || pbox == nil {
		t.Fatal("no paragraph found in input text")
	}
	// TODO we need to move container children from tree.Node to a newliy
	// supplied context. We changed the boxing-code to having context injeced from
	// extern. The code below will crash!
	t.Fatal("missing processing step of context injection")
	para, _, err := paragraphTextFromBox(pbox)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if para.Raw().IsVoid() {
		t.Errorf("inner text of para is void, should not be")
	}
	t.Logf("inner text = (%s)", para.Raw().String())
	//t.Logf("levels = %v", para.levels)
	//
	//f := cordsdotty(cords.Cord(para.Text.Styles()), t)
	// f := cordsdotty(para.Text.Raw(), t)
	// defer f.Close()
	t.Fail()
}

// ---------------------------------------------------------------------------

func buildTestDOM(hh string, t *testing.T) *dom.W3CNode {
	h, err := html.Parse(strings.NewReader(hh))
	if err != nil {
		t.Errorf("Cannot create test document")
	}
	dom := dom.FromHTMLParseTree(h, nil) // nil = no external stylesheet
	if dom == nil {
		t.Errorf("Could not build DOM from HTML")
	}
	return dom
}

func checkBoxTree(boxes frame.Container, err error, t *testing.T) {
	if err != nil {
		t.Fatalf(err.Error())
	} else if boxes == nil {
		t.Fatalf("Render tree root is null")
	} else {
		t.Logf("root node is %s", boxes.DOMNode().NodeName())
		if boxes.DOMNode().NodeName() != "#document" {
			t.Errorf("name of root element expected to be '#document")
		}
	}
	t.Logf("root node = %+v", boxes)
}

func findParaContainer(boxes frame.Container, t *testing.T) (bool, frame.Container) {
	switch boxes.Type() {
	case boxtree.TypePrincipal:
		if boxes.DOMNode().NodeName() == "p" {
			return true, boxes
		}
	}
	ch := boxes.TreeNode().Children(true)
	for _, c := range ch {
		subc := c.Payload.(frame.Container)
		found, p := findParaContainer(subc, t)
		if found {
			return true, p
		}
	}
	return false, nil
}

func recursiveContextAndPreset(c frame.Container, t *testing.T) {
	ctx := newContext(c, true)
	if ctx == nil {
		t.Error("no context created")
	}
	c.SetContext(ctx)
	c.PresetContained()
	ch := c.TreeNode().Children(true)
	for _, c := range ch {
		subc := c.Payload.(frame.Container)
		recursiveContextAndPreset(subc, t)
	}
}

func children(c frame.Container, t *testing.T) []frame.Container {
	if c.Context() != nil {
		// This is for the layout tree instead of the box tree:
		// instead of iterating over tree children, iterate over context children
		return c.Context().Contained()
	}
	kids := make([]frame.Container, 0, 16)
	n := c.TreeNode()
	for i := 0; i < n.ChildCount(); i++ {
		ch, ok := n.Child(i)
		if ok {
			kids = append(kids, ch.Payload.(frame.Container))
		} else {
			t.Errorf("cannot retrieve child #%d from component [%v]", i, boxtree.ContainerName(c))
		}
	}
	return kids
}

// --- test formatting context ------------------------------------------

type testContext struct {
	typ frame.FormattingContextType
	frame.ContextBase
	containers []frame.Container
}

func newContext(c frame.Container, isRoot bool) *testContext {
	ctx := &testContext{ContextBase: frame.MakeContextBase()}
	ctx.typ = frame.FormattingContextType(1)
	ctx.IsRootCtx = isRoot
	ctx.C = c
	ctx.Payload = ctx
	return ctx
}

func (ctx *testContext) Type() frame.FormattingContextType {
	return ctx.typ
}

func (ctx *testContext) AddContained(c frame.Container) {
	ctx.containers = append(ctx.containers, c)
}

func (ctx *testContext) Contained() []frame.Container {
	return ctx.containers
}

func (ctx testContext) Layout(*frame.FlowRoot) error {
	return fmt.Errorf("test context cannot layout()")
}

func (ctx testContext) Measure() (frame.Size, css.DimenT, css.DimenT) {
	return frame.Size{}, css.SomeDimen(100), css.SomeDimen(400)
}

// --- GraphViz dot output ---------------------------------------------------

/*
func dotty(doc *dom.W3CNode, t *testing.T) {
	if !dot {
		return
	}
	tmpfile, err := ioutil.TempFile(".", "dom.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name()) // clean up
	}()
	fmt.Printf("writing digraph to %s\n", tmpfile.Name())
	domdbg.ToGraphViz(doc, tmpfile, nil)
	outOption := fmt.Sprintf("-o%s.svg", tmpfile.Name())
	cmd := exec.Command("dot", "-Tsvg", outOption, tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("writing SVG tree image to tree.svg\n")
	if err := cmd.Run(); err != nil {
		t.Error(err.Error())
	}
}
*/

func cordsdotty(text cords.Cord, t *testing.T) {
	if !dot {
		return
	}
	tmpfile, err := ioutil.TempFile(".", "cord.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		tmpfile.Close()
		os.Remove(tmpfile.Name()) // clean up
	}()
	t.Logf("writing Cord digraph to %s\n", tmpfile.Name())
	cords.Cord2Dot(text, tmpfile)
	outOption := fmt.Sprintf("-o%s.svg", tmpfile.Name())
	cmd := exec.Command("dot", "-Tsvg", outOption, tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	t.Log("writing SVG cord tree\n")
	if err := cmd.Run(); err != nil {
		t.Error(err.Error())
	}
}
