package framedebug

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/npillmayer/schuko/gtrace"

	"github.com/npillmayer/tyse/engine/dom/w3cdom"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
)

// Parameters for GraphViz drawing.
type graphParamsType struct {
	Fontname    string
	StyleGroups []string
	BoxTmpl     *template.Template
	EdgeTmpl    *template.Template
}

// ToGraphViz creates a graphical representation of a render tree.
// It produces a DOT file format suitable as input for Graphviz, given a Writer.
func ToGraphViz(boxroot *boxtree.PrincipalBox, w io.Writer) {
	header, err := template.New("renderTree").Parse(graphHeadTmpl)
	if err != nil {
		panic(err)
	}
	gparams := graphParamsType{Fontname: "Helvetica"}
	gparams.BoxTmpl, _ = template.New("box").Funcs(
		template.FuncMap{
			"shortstring": shortText,
		}).Parse(boxTmpl)
	gparams.EdgeTmpl = template.Must(template.New("boxedge").Parse(edgeTmpl))
	err = header.Execute(w, gparams)
	if err != nil {
		panic(err)
	}
	dict := make(map[boxtree.Container]string, 4096)
	boxes(boxroot, w, dict, &gparams)
	w.Write([]byte("}\n"))
}

var cnt int

func boxes(c boxtree.Container, w io.Writer, dict map[boxtree.Container]string, gparams *graphParamsType) {
	cnt++
	if cnt == 300 {
		return
	}
	box(c, w, dict, gparams)
	gtrace.EngineTracer.Infof("container = %v", c)
	n := c.TreeNode()
	if n.ChildCount() >= 0 {
		children := n.Children()
		nn := n.ChildCount()
		gtrace.EngineTracer.Errorf("container has %d/%d children ..............", len(children), nn)

		//for i, ch := range children {
		for i := 0; i < n.ChildCount(); i++ {
			ch, ok := n.Child(i)
			if !ok {
				gtrace.EngineTracer.Errorf("Child at #%d could not be retrieved", i)
			} else {
				if ch == nil {
					gtrace.EngineTracer.Errorf("Child at #%d is nil", i)
				} else {
					gtrace.EngineTracer.Errorf("Child is %v", ch)
					child := ch.Payload.(boxtree.Container)
					gtrace.EngineTracer.Infof("  child[%d] = %v", i, child)
					boxes(child, w, dict, gparams)
					edge(c, child, w, dict, gparams)
				}
			}
		}
	}
}

func box(c boxtree.Container, w io.Writer, dict map[boxtree.Container]string, gparams *graphParamsType) {
	name := dict[c]
	if name == "" {
		sz := len(dict) + 1
		name = fmt.Sprintf("node%05d", sz)
		dict[c] = name
	}
	if err := gparams.BoxTmpl.Execute(w, &cbox{c, c.DOMNode(), name}); err != nil {
		panic(err)
	}
}

// Helper struct
type cbox struct {
	C    boxtree.Container
	N    w3cdom.Node
	Name string
}

func shortText(box *cbox) string {
	txt := box.N.NodeValue()
	disp := box.C.DisplayMode()
	sym := disp.Symbol()
	s := fmt.Sprintf("\"%s\u2000\\\"", sym)
	if len(txt) > 10 {
		s += txt[:10] + "…\\\"\""
	} else {
		s += txt + "\\\"\""
	}
	s = strings.Replace(s, "\n", `\\n`, -1)
	s = strings.Replace(s, "\t", `\\t`, -1)
	s = strings.Replace(s, " ", "\u2423", -1)
	return s
}

type cedge struct {
	N1, N2 cbox
}

func edge(c1 boxtree.Container, c2 boxtree.Container, w io.Writer, dict map[boxtree.Container]string,
	gparams *graphParamsType) {
	//
	//fmt.Printf("dict has %d entries\n", len(dict))
	name1 := dict[c1]
	name2 := dict[c2]
	e := cedge{cbox{c1, c1.DOMNode(), name1}, cbox{c2, c2.DOMNode(), name2}}
	if err := gparams.EdgeTmpl.Execute(w, e); err != nil {
		panic(err)
	}
}

// ---------------------------------------------------------------------------

func PrincipalLabel(pbox *boxtree.PrincipalBox) string {
	if pbox == nil {
		return "<empty box>"
	}
	name := pbox.DOMNode().NodeName()
	innerSym := pbox.DisplayMode().Symbol()
	//outerSym := pbox.outerMode.Symbol()
	outerSym := frame.NoMode.Symbol()
	if pbox.Context() != nil {
		if pbox.Context().Type() == boxtree.BlockFormattingContext {
			outerSym = frame.BlockMode.Symbol()
		} else {
			outerSym = frame.InlineMode.Symbol()
		}
	}
	//return fmt.Sprintf("%s %s %s", outerSym, innerSym, name)
	return fmt.Sprintf("%s %s %s", outerSym, innerSym, name)
}

func String(anon *boxtree.AnonymousBox) string {
	if anon == nil {
		return "<empty anon box>"
	}
	innerSym := anon.DisplayMode().Inner().Symbol()
	outerSym := anon.DisplayMode().Outer().Symbol()
	return fmt.Sprintf("%s %s", outerSym, innerSym)
}

// --- Templates --------------------------------------------------------

const graphHeadTmpl = `digraph g {                                                                                                             
  graph [labelloc="t" label="" splines=true overlap=false rankdir = "LR"];
  graph [{{ .Fontname }} = "helvetica" fontsize=12] ;
   node [fontname = "{{ .Fontname }}" fontsize=12] ;
   edge [fontname = "{{ .Fontname }}" fontsize=12] ;
`

const boxTmpl = `{{ if .C.IsAnonymous }}
{{ if .C.IsText }}
{{ .Name }}	[ label={{ shortstring . }} shape=box style=filled fillcolor=grey95 fontname="Courier" fontsize=11.0 ] ;
{{ else }}
{{ .Name }}	[ label="{{.C.String }}" shape=box style=filled fillcolor=grey90 fontname="Courier" fontsize=11.0 ] ;
{{ end }}
{{ else }}
{{ .Name }}	[ label={{ printf "%q" .C.String }} shape=box style=filled fillcolor=lightblue3 ] ;
{{ end }}
`

//const domEdgeTmpl = `{{ .N1.Name }} -> {{ .N2.Name }} [dir=none weight=1] ;
const edgeTmpl = `{{ .N1.Name }} -> {{ .N2.Name }} [weight=1] ;
`
