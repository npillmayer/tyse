package khipu

import (
	"strings"
	"testing"

	"github.com/npillmayer/tyse/engine/dom"
	"golang.org/x/net/html"
)

var myhtml = `
	<!DOCTYPE html>
	<html>
	<body>
	<h1>My First Heading</h1>
	<p>My short <b>first</b> paragraph.</p>
	</body>
	</html> 
`

// TestParaBreak: Build a DOM from a small input HTML string, use XPath to navigate
// to the only paragraph, extract the styled text of the paragraph, and encode
// it into a khipu using a monospace shaper.
/*
func TestParaBreak(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	// gtrace.EngineTracer = gologadapter.New()
	// gtrace.CoreTracer = gologadapter.New()
	// gtrace.EngineTracer.SetTraceLevel(tracing.LevelDebug)
	// gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	grapheme.SetupGraphemeClasses()
	root := buildDOM(myhtml, t)
	t.Logf("DOM ok")
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
	if len(k.knots) != 15 {
		t.Errorf("expected 15 knots in khipu, got %d", len(k.knots))
	}
}
*/

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
