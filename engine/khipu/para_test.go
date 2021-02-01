package khipu

import (
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing/gologadapter"
	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/khipu/styled"
	"github.com/npillmayer/tyse/engine/text/monospace"
	"github.com/npillmayer/uax/grapheme"
	"golang.org/x/net/html"
)

var myhtml = `
	<!DOCTYPE html>
	<html>
	<body>
	<h1>My First Heading</h1>
	<p>My <b>first</b> paragraph.</p>
	</body>
	</html> 
`

func TestBasic(t *testing.T) {
	//teardown := testconfig.QuickConfig(t)
	//defer teardown()
	gtrace.CoreTracer = gologadapter.New()
	//
	grapheme.SetupGraphemeClasses()
	root := buildDOM(myhtml, t)
	xp := root.XPath()
	n, err := xp.FindOne("//p")
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("n=%v", n)
	if n == nil {
		t.Fatal("p not found")
	}
	p, _ := dom.NodeFromTreeNode(n)
	// if err != nil {
	// 	t.Fatalf(err.Error())
	// }
	para, err := styled.InnerParagraphText(p)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("inner text of DOM = '%s'", para.Raw().String())
	k, err := EncodeParagraph(para, 0, monospace.Shaper(11*dimen.PT, nil), nil, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if k == nil {
		t.Fatalf("resulting khipu is nil, should not be")
	}
	t.Logf("khipu = %v", k)
	t.Fail()
}

// --- Helpers ---------------------------------------------------------------

func buildDOM(hh string, t *testing.T) *dom.W3CNode {
	h, err := html.Parse(strings.NewReader(hh))
	if err != nil {
		t.Errorf("Cannot create test document")
	}
	dom := dom.FromHTMLParseTree(h, nil) // nil = no external stylesheet
	if dom == nil {
		t.Fatalf("Could not build DOM from HTML")
	}
	return dom
}

func findPara(dom *dom.W3CNode) {
	//
}
