package boxtree_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/domdbg"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
	"github.com/npillmayer/tyse/engine/frame/framedebug"
	"golang.org/x/net/html"
)

func TestCSSAttributing(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.frame.box")
	defer teardown()
	//
	domroot := buildDOM(t, false)
	boxes, err := boxtree.BuildBoxTree(domroot)
	checkBoxTree(boxes, err, t)
	//dottyBoxTree(boxes, t)
}

// ---------------------------------------------------------------------------

var minihtml = `
<html><head>
<style>
  body { border-color: red; }
</style>
</head><body>
  <p>The quick brown fox jumps over the lazy</p><b>dog.</b>
  <p id="world">Hello <b>World</b>!</p>
  <p style="padding-left: 5px; position: fixed;">This is a test.</p>
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

func dottyBoxTree(root frame.Container, t *testing.T) *os.File {
	tmpfile, err := ioutil.TempFile(".", "boxtree.*.dot")
	if err != nil {
		log.Fatal(err)
	}
	//defer os.Remove(tmpfile.Name()) // clean up
	fmt.Printf("writing BoxTree to %s\n", tmpfile.Name())
	framedebug.ToGraphViz(root.(*boxtree.PrincipalBox), tmpfile)
	defer tmpfile.Close()
	cmd := exec.Command("dot", "-Tsvg", "-oboxtree.svg", tmpfile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("writing SVG BoxTree image to boxtree.svg\n")
	if err := cmd.Run(); err != nil {
		t.Error(err.Error())
	}
	fmt.Printf("done with boxtree.svg\n")
	return tmpfile
}
