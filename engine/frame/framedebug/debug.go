package framedebug

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/npillmayer/schuko/tracing"

	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/dom/w3cdom"
	"github.com/npillmayer/tyse/engine/frame"
	"github.com/npillmayer/tyse/engine/frame/boxtree"
)

// Parameters for GraphViz drawing.
type graphParamsType struct {
	Fontname    string
	StyleGroups []string
	BoxTmpl     *template.Template
	PBoxTmpl    *template.Template
	EdgeTmpl    *template.Template
	cnt         int
}

// ToGraphViz creates a graphical representation of a render tree.
// It produces a DOT file format suitable as input for Graphviz, given a Writer.
func ToGraphViz(boxroot *boxtree.PrincipalBox, w io.Writer, tracer tracing.Trace) {
	header, err := template.New("renderTree").Parse(graphHeadTmpl)
	if err != nil {
		panic(err)
	}
	gparams := graphParamsType{Fontname: "Helvetica"}
	gparams.BoxTmpl, _ = template.New("box").Funcs(
		template.FuncMap{
			"shortstring": shortText,
			"istext":      isTextBox,
			"label":       label,
		}).Parse(boxTmpl)
	gparams.PBoxTmpl, _ = template.New("pbox").Funcs(
		template.FuncMap{
			"shortstring": shortText,
			"istext":      isTextBox,
			"label":       label,
		}).Parse(pboxTmpl)
	gparams.EdgeTmpl = template.Must(template.New("boxedge").Parse(edgeTmpl))
	err = header.Execute(w, gparams)
	if err != nil {
		panic(err)
	}
	dict := make(map[frame.Container]string, 4096)
	boxes(boxroot, w, dict, &gparams, tracer)
	w.Write([]byte("}\n"))
}

func boxes(c frame.Container, w io.Writer, dict map[frame.Container]string, gparams *graphParamsType,
	tracer tracing.Trace) {
	//
	gparams.cnt++
	if gparams.cnt == 300 {
		return // guard against errorneous cycles
	}
	box(c, w, dict, gparams)
	tracer.Debugf("container = %v", c)
	kids := children(c)
	for i, child := range kids {
		tracing.Debugf("  child[%d] = %v", i, child)
		boxes(child, w, dict, gparams, tracer)
		edge(c, child, w, dict, gparams)
	}
}

func children(c frame.Container) []frame.Container {
	if c.Context() != nil {
		// This is for the layout tree instead of the box tree:
		// instead of iterating over tree children, iterate over context children
		return c.Context().Contained()
	}
	kids := make([]frame.Container, 0, 16)
	n := c.TreeNode()
	for i := 0; i < n.ChildCount(); i++ {
		ch, ok := n.Child(i)
		if !ok {
			tracing.Debugf("Child at #%d could not be retrieved", i)
		} else if ch == nil {
			tracing.Debugf("Child at #%d is nil", i)
		} else {
			kids = append(kids, ch.Payload.(frame.Container))
		}
	}
	return kids
}

func box(c frame.Container, w io.Writer, dict map[frame.Container]string, gparams *graphParamsType) {
	name := dict[c]
	if name == "" {
		sz := len(dict) + 1
		name = fmt.Sprintf("node%05d", sz)
		dict[c] = name
	}
	if p, ok := c.(*boxtree.PrincipalBox); ok {
		if b := styledBoxParams(p, name); b != nil {
			if err := gparams.PBoxTmpl.Execute(w, b); err != nil {
				panic(err)
			}
		} else if err := gparams.BoxTmpl.Execute(w, &cbox{c, c.DOMNode(), name}); err != nil {
			panic(err)
		}
	} else {
		if err := gparams.BoxTmpl.Execute(w, &cbox{c, c.DOMNode(), name}); err != nil {
			panic(err)
		}
	}
}

// Helper structs
type cbox struct {
	C    frame.Container
	N    w3cdom.Node
	Name string
}

type pbox struct {
	C      *boxtree.PrincipalBox
	N      w3cdom.Node
	Name   string
	Color  string
	Fill   string
	Border string
}

func shortText(box *cbox) string {
	txt := box.N.NodeValue()
	s := fmt.Sprintf("\"%s\u2000\\\"", "T")
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

func edge(c1 frame.Container, c2 frame.Container, w io.Writer, dict map[frame.Container]string,
	gparams *graphParamsType) {
	//
	name1 := dict[c1]
	name2 := dict[c2]
	e := cedge{cbox{c1, c1.DOMNode(), name1}, cbox{c2, c2.DOMNode(), name2}}
	if err := gparams.EdgeTmpl.Execute(w, e); err != nil {
		panic(err)
	}
}

// ---------------------------------------------------------------------------

func label(c frame.Container) string {
	switch b := c.(type) {
	case *boxtree.PrincipalBox:
		return "\"" + PrincipalLabel(b) + "\""
	case *boxtree.AnonymousBox:
		return "\"" + AnonLabel(b) + "\""
	}
	return "\"?\""
}

func PrincipalLabel(pbox *boxtree.PrincipalBox) string {
	if pbox == nil {
		return "<empty box>"
	}
	name := pbox.DOMNode().NodeName()
	innerSym := pbox.DisplayMode().Inner().Symbol()
	outerSym := pbox.DisplayMode().Outer().Symbol()
	return fmt.Sprintf("%s %s %s", outerSym, innerSym, name)
}

func AnonLabel(c frame.Container) string {
	if c == nil {
		return "<empty anon box>"
	}
	innerSym := c.DisplayMode().Inner().Symbol()
	outerSym := c.DisplayMode().Outer().Symbol()
	return fmt.Sprintf("%s %s", outerSym, innerSym)
}

func isTextBox(c frame.Container) bool {
	_, ok := c.(*boxtree.TextBox)
	return ok
}

// --- Templates --------------------------------------------------------

const graphHeadTmpl = `digraph g {                                                                                                             
  graph [labelloc="t" label="" splines=true overlap=false rankdir = "LR"];
  graph [{{ .Fontname }} = "helvetica" fontsize=12] ;
   node [fontname = "{{ .Fontname }}" fontsize=12] ;
   edge [fontname = "{{ .Fontname }}" fontsize=12] ;
`
const boxTmpl = `{{ if istext .C }}
{{ .Name }}	[ label={{ shortstring . }} shape=box style=filled fillcolor=grey95 fontname="Courier" fontsize=11.0 ] ;
{{ else }}
{{ .Name }}	[ label={{ label .C }} shape=box style=filled fillcolor=lightblue3 ] ;
{{ end }}
`

const pboxTmpl = `
{{ .Name }}	[ label={{ label .C }} shape=box style=filled {{ .Fill }} {{ .Color }} {{ .Border }}] ;
`

const nouseboxTmpl = `{{ if .C.IsAnonymous }}
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

func styledBoxParams(p *boxtree.PrincipalBox, name string) (b *pbox) {
	isFixed := false
	if p.CSSBox().HasFixedBorderBoxWidth(true) {
		isFixed = true
	}
	if p.Box.Styles == nil && isFixed {
		return nil // we'll create a standard box
	}
	if p.Box.Styles == nil {
		b = &pbox{
			C:     p,
			N:     p.DOMNode(),
			Name:  name,
			Color: "color=black",
			Fill:  "fillcolor=lightblue3",
		}
	} else {
		sty := p.Box.Styles
		b = &pbox{
			C:     p,
			N:     p.DOMNode(),
			Name:  name,
			Color: fmt.Sprintf("color=\"%s\"", style.ColorString(sty.Border.LineColor)),
			Fill:  fmt.Sprintf("fillcolor=\"%s\"", style.ColorString(sty.Colors.Background)),
		}
	}
	if p.CSSBox().HasFixedBorderBoxWidth(true) {
		b.Border = ""
	} else {
		b.Border = "peripheries=2"
	}
	return b
}
