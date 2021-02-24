package xpath_test

import (
	"strings"
	"testing"

	"github.com/aymerick/douceur/parser"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/logrusadapter"
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/dom/style/cssom"
	"github.com/npillmayer/tyse/engine/dom/style/cssom/douceuradapter"
	"github.com/npillmayer/tyse/engine/dom/styledtree"
	"github.com/npillmayer/tyse/engine/dom/styledtree/xpathadapter"
	"github.com/npillmayer/tyse/engine/dom/xpath"
	"github.com/npillmayer/tyse/engine/tree"
	"golang.org/x/net/html"
)

var T tracing.Trace

const (
	html1 string = `<body><p class="hello">Hello World</p></body>`
	html2 string = `<body><p id="single">Hello</p><p>World</p></body>`
	html3 string = `<body><p>Links:</p><ul><li><a href="foo">Foo</a><li>
<a href="/bar/baz">BarBaz</a></ul></body>`
	css1 string = `p { padding: 10px; } p.hello { color: blue; } #single { margin: 7px; }`
)

func Test0(t *testing.T) {
	gtrace.EngineTracer = logrusadapter.New()
	gtrace.EngineTracer.SetTraceLevel(tracing.LevelDebug)
	T = gtrace.EngineTracer
}

type UNUSED interface{}

func Test1(t *testing.T) {
	_, tree := setupTest(html1, css1)
	if tree == nil {
		t.Error("failed to setup test")
	}
	paras := findNodesFor("p", nil, &tree.Node)
	if !assertProperty(paras, "padding-top").equals("10px") {
		t.Error("padding-top of paragraphs should be 10px")
	}
}

func Test2(t *testing.T) {
	q, tree := setupTest(html1, css1)
	if tree == nil {
		t.Error("failed to setup test")
	}
	paras := findNodesFor("p.hello", q, &tree.Node)
	if !assertProperty(paras, "color").equals("blue") {
		t.Error("color of paragraph with class=hello should be blue")
	}
}

// --- Helpers ----------------------------------------------------------

func getTestDOM(s string) *html.Node {
	doc, _ := html.Parse(strings.NewReader(s))
	return doc
}

func getTestCSS(s string) cssom.StyleSheet {
	css, _ := parser.Parse(s)
	return douceuradapter.Wrap(css)
}

func setupTest(htmlStr string, cssStr string) (*UNUSED, *styledtree.StyNode) {
	dom := getTestDOM(htmlStr)
	css := getTestCSS(cssStr)
	styler := cssom.NewCSSOM(nil)
	styler.AddStylesForScope(nil, css, cssom.Author)
	styledTree, err := styler.Style(dom, styledtree.Creator())
	if err != nil {
		T.Errorf("error: %s", err)
		return nil, nil
	}
	//doc := goquery.NewDocumentFromNode(dom)
	return nil, styledtree.Node(styledTree)
}

func findNodesFor(xpstr string, doc UNUSED, tree *tree.Node) []*tree.Node {
	nav := xpathadapter.NewNavigator(styledtree.Node(tree))
	xp, _ := xpath.NewXPath(nav, xpathadapter.CurrentNode)
	nodes, _ := xp.Find(xpstr)
	T.Debugf("found styled nodes: %v", nodes)
	return nodes
}

type props []style.Property

func assertProperty(nodes []*tree.Node, key string) props {
	if nodes == nil {
		return nil
	}
	var pp props
	sncreat := styledtree.Creator()
	for _, sn := range nodes {
		p, _ := style.GetCascadedProperty(sn, key, sncreat.ToStyler)
		T.Debugf("property %s of %s = %s", key, sn, p)
		pp = append(pp, p)
	}
	return pp
}

func (pp props) equals(property style.Property) bool {
	for _, p := range pp {
		if p != property {
			return false
		}
	}
	return true
}
