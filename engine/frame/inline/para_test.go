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
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/domdbg"
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

func TestParaCreate(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.EngineTracer.SetTraceLevel(tracing.LevelDebug)
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelInfo)
	//
	domroot := buildTestDOM(testhtml, t)
	boxes, err := boxtree.BuildBoxTree(domroot)
	checkBoxTree(boxes, err, t)
	found, pbox := findParaContainer(boxes, t)
	if !found || pbox == nil {
		t.Fatal("no paragraph found in input text")
	}
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

func dotty(doc *dom.W3CNode, t *testing.T) *os.File {
	tmpfile, err := ioutil.TempFile(".", "cord.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(tmpfile.Name()) // clean up
	fmt.Printf("writing digraph to %s\n", tmpfile.Name())
	domdbg.ToGraphViz(doc, tmpfile, nil)
	cmd := exec.Command("dot", "-Tsvg", "-otree.svg", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("writing SVG tree image to tree.svg\n")
	if err := cmd.Run(); err != nil {
		t.Error(err.Error())
	}
	return tmpfile
}

func cordsdotty(text cords.Cord, t *testing.T) *os.File {
	tmpfile, err := ioutil.TempFile(".", "cord.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(tmpfile.Name()) // clean up
	fmt.Printf("writing digraph to %s\n", tmpfile.Name())
	cords.Cord2Dot(text, tmpfile)
	cmd := exec.Command("dot", "-Tsvg", "-ocordtree.svg", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("writing SVG cord tree to cordtree.svg\n")
	if err := cmd.Run(); err != nil {
		t.Error(err.Error())
	}
	return tmpfile
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
