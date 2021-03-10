package layout

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/domdbg"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
	"github.com/npillmayer/tyse/engine/frame/framedebug"
	"golang.org/x/net/html"
)

func TestLayout(t *testing.T) {
	// teardown := testconfig.QuickConfig(t)
	// defer teardown()
	gtrace.EngineTracer = gologadapter.New()
	gtrace.EngineTracer.SetTraceLevel(tracing.LevelDebug)
	gtrace.CoreTracer = gologadapter.New()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelError)
	//
	domroot := buildDOM(t, false)
	boxes, err := boxtree.BuildBoxTree(domroot)
	checkBoxTree(boxes, err, t)
	v := View{Width: 8 * dimen.CM}
	r := BoxTreeToLayoutTree(boxes.(*boxtree.PrincipalBox), &v)
	t.Logf("resulting W = %s", r.W)
	if r.lastErr != nil {
		t.Errorf("resulting error is: %v", r.lastErr)
	}
	//dottyLayoutTree(boxes, t)
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
	tl := gtrace.EngineTracer.GetTraceLevel()
	gtrace.EngineTracer.SetTraceLevel(tracing.LevelError)
	//
	tmpfile, err := ioutil.TempFile(".", "dom.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(tmpfile.Name()) // clean up
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
	gtrace.EngineTracer.SetTraceLevel(tl)
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

func dottyLayoutTree(root frame.Container, t *testing.T) *os.File {
	tl := gtrace.EngineTracer.GetTraceLevel()
	gtrace.EngineTracer.SetTraceLevel(tracing.LevelError)
	//
	tmpfile, err := ioutil.TempFile(".", "layouttree.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(tmpfile.Name()) // clean up
	fmt.Printf("writing BoxTree to %s\n", tmpfile.Name())
	framedebug.ToGraphViz(root.(*boxtree.PrincipalBox), tmpfile)
	defer tmpfile.Close()
	cmd := exec.Command("dot", "-Tsvg", "-olayouttree.svg", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("writing SVG BoxTree image to layouttree.svg\n")
	if err := cmd.Run(); err != nil {
		t.Error(err.Error())
	}
	fmt.Printf("done with layouttree.svg\n")
	gtrace.EngineTracer.SetTraceLevel(tl)
	return tmpfile
}
