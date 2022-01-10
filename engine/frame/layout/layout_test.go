package layout

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/domdbg"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
	"github.com/npillmayer/tyse/engine/frame/framedebug"
	"golang.org/x/net/html"
)

func TestLayout(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.frame")
	defer teardown()
	dottyTracer := gotestingadapter.New(t)
	dottyTracer.SetTraceLevel(tracing.LevelError)
	//
	domroot := buildDOM(t, false)
	boxes, err := boxtree.BuildBoxTree(domroot)
	checkBoxTree(boxes, err, t)
	v := View{Width: 8 * dimen.CM}
	r := BoxTreeToLayoutTree(boxes.RenderNode().(*boxtree.PrincipalBox), &v)
	if r.lastErr != nil {
		t.Errorf("layout tree: resulting error is: %v", r.lastErr)
	}
	if r.W == 0 {
		t.Logf("resulting W = %s", r.W)
		t.Errorf("layout tree: resulting bounding box has no width")
	}
	dottyLayoutTree(boxes, t, dottyTracer)
}

// ---------------------------------------------------------------------------

var minihtml = `
<html><head>
<style>
  p { border-color: red; }
</style>
</head><body>
  <p>The quick brown fox jumps over the lazy dog.</p>
</body>
`

func buildDOM(t *testing.T, drawit bool) *dom.W3CNode {
	h, err := html.Parse(strings.NewReader(minihtml))
	if err != nil {
		t.Errorf("Cannot create test document")
	}
	dom := dom.FromHTMLParseTree(h, nil) // nil = no external stylesheet
	if dom == nil {
		t.Fatal("Could not build DOM from HTML")
	} else if drawit {
		dottyDOM(dom, t)
	}
	return dom
}

func dottyDOM(doc *dom.W3CNode, t *testing.T) *os.File {
	tmpfile, err := ioutil.TempFile(".", "dom.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up
	fmt.Printf("writing DOM to %s\n", tmpfile.Name())
	domdbg.ToGraphViz(doc, tmpfile, nil)
	defer tmpfile.Close()
	cmd := exec.Command("dot", "-Tsvg", "-odom.svg", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("writing SVG DOM image to dom.svg\n")
	if err := cmd.Run(); err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("done with dom.svg\n")
	return tmpfile
}

func checkBoxTree(boxes *frame.ContainerBase, err error, t *testing.T) {
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

func dottyLayoutTree(root *frame.ContainerBase, t *testing.T, tracer tracing.Trace) *os.File {
	tmpfile, err := ioutil.TempFile(".", "layouttree.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(tmpfile.Name()) // clean up
	fmt.Printf("writing BoxTree to %s\n", tmpfile.Name())
	framedebug.ToGraphViz(root.RenderNode().(*boxtree.PrincipalBox), tmpfile, tracer)
	defer tmpfile.Close()
	cmd := exec.Command("dot", "-Tsvg", "-olayouttree.svg", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("writing SVG BoxTree image to layouttree.svg\n")
	if err := cmd.Run(); err != nil {
		t.Error(err.Error())
	}
	fmt.Printf("done with layouttree.svg\n")
	return tmpfile
}
