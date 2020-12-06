/*
Package domdbg implements helpers to debug a DOM tree.

BSD License

Copyright (c) 2017–20, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of Norbert Pillmayer nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.  */
package domdbg

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/cssom/style"
	"golang.org/x/net/html"
)

// Parameters for GraphViz drawing.
type graphParamsType struct {
	Fontname       string
	StyleGroups    []string
	NodeTmpl       *template.Template
	EdgeTmpl       *template.Template
	StylegroupTmpl *template.Template
	PgedgeTmpl     *template.Template
	PgpgTmpl       *template.Template
}

var defaultGroups = []string{
	style.PGMargins,
	style.PGPadding,
	style.PGBorder,
	style.PGDisplay,
}

// ToGraphViz outputs a diagram for a DOM tree. The diagram is in
// GraphViz (DOT) format. Clients have to provide the root node of
// the DOM, a Writer, and an optional list of style parameter groups.
// The diagram will include all styles belonging to one of the
// parameter groups.
// If the client does not provide a list of style groups, the following
// default will be used:
//     - Margins
//     - Padding
//     - Border
//     - Display
func ToGraphViz(doc *dom.W3CNode, w io.Writer, styleGroups []string) {
	tmpl, err := template.New("dom").Parse(graphHeadTmpl)
	if err != nil {
		panic(err)
	}
	gparams := graphParamsType{Fontname: "Helvetica"}
	gparams.NodeTmpl, _ = template.New("domnode").Funcs(
		template.FuncMap{
			"shortstring": shortText,
		}).Parse(domNodeTmpl)
	gparams.EdgeTmpl = template.Must(template.New("domedge").Parse(domEdgeTmpl))
	gparams.StylegroupTmpl = template.Must(template.New("stylegroup").Parse(styleGroupTmpl))
	gparams.PgedgeTmpl = template.Must(template.New("pgedge").Parse(pgEdgeTmpl))
	gparams.PgpgTmpl = template.Must(template.New("pgpgedge").Parse(pgpgEdgeTmpl))
	gparams.StyleGroups = styleGroups
	if styleGroups == nil {
		gparams.StyleGroups = defaultGroups
	}
	err = tmpl.Execute(w, gparams)
	if err != nil {
		panic(err)
	}
	dict := make(map[*html.Node]string, 4096)
	nodes(doc, w, dict, &gparams)
	w.Write([]byte("}\n"))
}

type node struct {
	N    *dom.W3CNode
	Name string
}

func nodes(n *dom.W3CNode, w io.Writer, dict map[*html.Node]string, gparams *graphParamsType) {
	domNode(n, w, dict, gparams)
	if n.HasChildNodes() {
		ch := n.FirstChild().(*dom.W3CNode)
		for ch != nil {
			nodes(ch, w, dict, gparams)
			domEdge(n, ch, w, dict, gparams)
			c := ch.NextSibling()
			if c != nil {
				ch = c.(*dom.W3CNode)
			} else {
				ch = nil
			}
		}
	}
}

func domNode(n *dom.W3CNode, w io.Writer, dict map[*html.Node]string, gparams *graphParamsType) {
	name := dict[n.HTMLNode()]
	if name == "" {
		l := len(dict) + 1
		name = fmt.Sprintf("node%05d", l)
		dict[n.HTMLNode()] = name
	}
	if err := gparams.NodeTmpl.Execute(w, &node{n, name}); err != nil {
		panic(err)
	}
	domStyles(n, w, dict, gparams)
}

func domStyles(n *dom.W3CNode, w io.Writer, dict map[*html.Node]string, gparams *graphParamsType) {
	pmap := n.ComputedStyles().Styles()
	var prev *style.PropertyGroup
	for _, s := range gparams.StyleGroups {
		pg := pmap.Group(s)
		if pg != nil {
			if err := gparams.StylegroupTmpl.Execute(w, pg); err != nil {
				panic(err)
			}
			if prev == nil {
				pgEdge(n, pg, w, dict, gparams)
			} else {
				pgpgEdge(prev, pg, w, dict, gparams)
			}
			prev = pg
		}
	}
}

type edge struct {
	N1, N2 node
}

func domEdge(n1 *dom.W3CNode, n2 *dom.W3CNode, w io.Writer, dict map[*html.Node]string,
	gparams *graphParamsType) {
	//
	//fmt.Printf("dict has %d entries\n", len(dict))
	name1 := dict[n1.HTMLNode()]
	name2 := dict[n2.HTMLNode()]
	e := edge{node{n1, name1}, node{n2, name2}}
	if err := gparams.EdgeTmpl.Execute(w, e); err != nil {
		panic(err)
	}
}

type pgedge struct {
	Name      string
	PropGroup *style.PropertyGroup
}

func pgEdge(n *dom.W3CNode, pg *style.PropertyGroup, w io.Writer, dict map[*html.Node]string,
	gparams *graphParamsType) {
	//
	name := dict[n.HTMLNode()]
	if err := gparams.PgedgeTmpl.Execute(w, pgedge{name, pg}); err != nil {
		panic(err)
	}
}

func pgpgEdge(pg1 *style.PropertyGroup, pg2 *style.PropertyGroup, w io.Writer,
	dict map[*html.Node]string, gparams *graphParamsType) {
	//
	if err := gparams.PgpgTmpl.Execute(w, []*style.PropertyGroup{pg1, pg2}); err != nil {
		panic(err)
	}
}

func shortText(n *dom.W3CNode) string {
	h := n.HTMLNode()
	s := "\"\\\""
	if len(h.Data) > 10 {
		s += h.Data[:10] + "...\\\"\""
	} else {
		s += h.Data + "\\\"\""
	}
	s = strings.Replace(s, "\n", `\\n`, -1)
	s = strings.Replace(s, "\t", `\\t`, -1)
	s = strings.Replace(s, " ", "\u2423", -1)
	return s
}

// --- Templates --------------------------------------------------------

const graphHeadTmpl = `digraph g {                                                                                                             
  graph [labelloc="t" label="" splines=true overlap=false rankdir = "LR"];
  graph [{{ .Fontname }} = "helvetica" fontsize=14] ;
   node [fontname = "{{ .Fontname }}" fontsize=14] ;
   edge [fontname = "{{ .Fontname }}" fontsize=14] ;
`

const domNodeTmpl = `{{ if eq .N.NodeName "#text" }}
{{ .Name }}	[ label={{ shortstring .N }} shape=box style=filled fillcolor=grey95 fontname="Courier" fontsize=11.0 ] ;
{{ else }}
{{ .Name }}	[ label={{ printf "%q" .N.NodeName }} shape=ellipse style=filled fillcolor=lightblue3 ] ;
{{ end }}
`

const styleGroupTmpl = `{{ printf "pg%p" . }} [ style="filled" penwidth=1 fillcolor="ivory3" shape="Mrecord" fontsize=12
    label=<<table border="0" cellborder="0" cellpadding="2" cellspacing="0" bgcolor="ivory3">
      <tr><td bgcolor="azure4" align="center" colspan="2"><font color="white">{{ .Name }}</font></td></tr>
      {{ range .Properties }}
      <tr><td align="right">{{ .Key }}:</td><td>{{ .Value }}</td></tr>
      {{ else }}
      <tr><td colspan="2">no styles</td></tr>
      {{ end }}
    </table>> ] ;
`

//const domEdgeTmpl = `{{ .N1.Name }} -> {{ .N2.Name }} [dir=none weight=1] ;
const domEdgeTmpl = `{{ .N1.Name }} -> {{ .N2.Name }} [weight=1] ;
`

const pgEdgeTmpl = `{{ .Name }} -> {{ printf "pg%p" .PropGroup }} [dir=none weight=1 style="dashed"] ;
`

const pgpgEdgeTmpl = `{{ index . 0 | printf "pg%p"  }} -> {{ index . 1 | printf "pg%p" }} [dir=none weight=1 style="dashed"] ;
`
