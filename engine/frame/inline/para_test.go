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
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	domroot := buildTestDOM(testhtml, t)
	text, err := InnerParagraphText(domroot)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if text.Raw().IsVoid() {
		t.Errorf("inner text of para is void, should not be")
	}
	t.Logf("inner text = (%s)", text.Raw().String())
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
