package khipu

import (
	"strings"
	"testing"

	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/khipu/styled"
	"github.com/npillmayer/tyse/engine/text"
	"github.com/npillmayer/tyse/engine/text/glypher"
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
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	//
	dom := buildDOM(myhtml, t)
	para, err := styled.InnerParagraphText(dom)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("inner text of DOM = '%s'", para.Raw().String())
	k, err := EncodeParagraph(para, 0, glypher.Instance(text.LeftToRight, text.Latin), nil, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if k == nil {
		t.Fatalf("resulting khipu is nil, should not be")
	}
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
