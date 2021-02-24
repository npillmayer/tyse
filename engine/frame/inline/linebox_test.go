package inline

import (
	"strings"
	"testing"

	"github.com/npillmayer/tyse/core/dimen"
	"github.com/npillmayer/tyse/core/parameters"
	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/frame/khipu"
	"github.com/npillmayer/tyse/engine/frame/khipu/linebreak"
	"github.com/npillmayer/tyse/engine/frame/khipu/linebreak/firstfit"
	"github.com/npillmayer/tyse/engine/glyphing/monospace"
	"golang.org/x/net/html"
)

var myhtml = `
	<!DOCTYPE html>
	<html>
	<body>
	<h1>A Heading</h1>
	<p>
Als Gregor Samsa eines Morgens aus unruhigen Träumen erwachte, fand er sich in seinem Bett zu einem
ungeheueren Ungeziefer verwandelt. Er lag auf seinem panzerartig harten Rücken und sah, wenn er den
Kopf ein wenig hob, seinen gewölbten, braunen, von bogenförmigen Versteifungen geteilten Bauch,
auf dessen Höhe sich die Bettdecke, zum gänzlichen Niedergleiten bereit, kaum noch erhalten konnte.
Seine vielen, im Vergleich zu seinem sonstigen Umfang kläglich dünnen Beine flimmerten ihm hilflos
vor den Augen.
	</p>
	</body>
	</html> 
`

func TestBox1(t *testing.T) {
	root := buildDOM(myhtml, t)
	p := findPara(root, t)
	para, err := InnerParagraphText(p)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("inner text of DOM = '%s'", para.Raw().String())
	regs := parameters.NewTypesettingRegisters()
	k, err := khipu.EncodeParagraph(para.Paragraph, 0, monospace.Shaper(11*dimen.PT, nil), nil, regs)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if k == nil {
		t.Fatalf("resulting khipu is nil, should not be")
	}
	t.Logf("khipu = %v", k)
	parshape := linebreak.RectangularParShape(80 * 10 * dimen.BP)
	cursor := linebreak.NewFixedWidthCursor(khipu.NewCursor(k), 10*dimen.BP, 0)
	breakpoints, err := firstfit.BreakParagraph(cursor, parshape, nil)
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("text broken up into %d lines", len(breakpoints))
	t.Logf("     |---------+---------+---------+---------+---------+---------+---------+---------|")
	j := int64(0)
	for i := 1; i < len(breakpoints); i++ {
		t.Logf("%3d: %s", i, k.Text(j, breakpoints[i].Position()))
		j = breakpoints[i].Position()
	}
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

func findPara(root *dom.W3CNode, t *testing.T) *dom.W3CNode {
	xp := root.XPath()
	n, err := xp.FindOne("//p")
	if err != nil {
		t.Fatalf(err.Error())
	}
	t.Logf("node = %v", n)
	if n == nil {
		t.Fatal("no paragraph found")
	}
	p, _ := dom.NodeFromTreeNode(n)
	return p
}
