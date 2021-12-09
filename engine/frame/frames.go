package frame

import (
	"fmt"
	"strings"

	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/dom/style"
	"github.com/npillmayer/tyse/engine/dom/style/css"
	"golang.org/x/net/html"
)

/*
BSD License

Copyright (c) 2017–2021, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of this software nor the names of its contributors
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
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

*/

// Typesetting in frames

// Regions:
// https://drafts.csswg.org/css-regions-1/
// http://cna.mamk.fi/Public/FJAK/MOAC_MTA_HTML5_App_Dev/c06.pdf

/* CSS:  https://github.com/ericchiang/css
   https://github.com/andybalholm/cascadia
   https://code.tutsplus.com/tutorials/the-30-css-selectors-you-must-memorize--net-16048
*/

// DisplayModeForDOMNode returns outer and inner display mode for a given DOM node.
func DisplayModeForDOMNode(domnode *dom.W3CNode) css.DisplayMode {
	if domnode == nil || domnode.HTMLNode() == nil {
		return css.NoMode
	}
	if domnode.NodeType() == html.TextNode {
		return css.InlineMode
	}
	display := domnode.ComputedStyles().GetPropertyValue("display")
	//T().Infof("property display = %v", display)
	if display.String() == "" || display.String() == "initial" {
		//outerMode, innerMode = DefaultDisplayModeForHTMLNode(domnode.HTMLNode())
		display = style.DisplayPropertyForHTMLNode(domnode.HTMLNode())
	}
	//outerMode, innerMode, err = ParseDisplay(display.String())
	mode, err := css.ParseDisplay(display.String())
	if err != nil {
		tracer().Errorf("unrecognized display property: %s", display)
		mode = css.BlockMode
	}
	//T().Infof("display modes = %s", mode)
	return mode
}

// DefaultDisplayModeForHTMLNode returns the default display mode for a HTML node type,
// as described by the CSS specification.
//
// TODO possibly move this to package style (= part of browser defaults)
// If, then return a string.
/*
func DefaultDisplayModeForHTMLNode(h *html.Node) (css.DisplayMode, css.DisplayMode) {
	if h == nil {
		return css.NoMode, css.NoMode
	}
	switch h.Type {
	case html.DocumentNode:
		return css.BlockMode, css.BlockMode
	case html.TextNode:
		return css.InlineMode, css.InlineMode
	case html.ElementNode:
		switch h.Data {
		case "table":
			return css.BlockMode, css.TableMode
		case "ul", "ol":
			return css.BlockMode, css.ListItemMode
		case "li":
			return css.ListItemMode, css.BlockMode
		case "html", "body", "div", "section", "article", "nav":
			return css.BlockMode, css.BlockMode
		case "p":
			return css.BlockMode, css.InlineMode
		case "span", "i", "b", "strong", "em":
			return css.InlineMode, css.InlineMode
		case "h1", "h2", "h3", "h4", "h5", "h6":
			return css.BlockMode, css.InlineMode
		default:
			return css.BlockMode, css.BlockMode
		}
	default:
		T().Errorf("Have styled node for non-element ?!?")
		T().Errorf(" type of node = %d", h.Type)
		T().Errorf(" name of node = %s", h.Data)
		T().Infof("unknown HTML element will stack children vertically")
		return css.BlockMode, css.BlockMode
	}
}
*/

// ---------------------------------------------------------------------------

func dbgNodeString(domnode *dom.W3CNode) string {
	if domnode == nil {
		return "DOM(null)"
	}
	return fmt.Sprintf("DOM(%s/%s)", domnode.NodeName(), shortText(domnode))
}

func shortText(n *dom.W3CNode) string {
	h := n.HTMLNode()
	s := "\""
	if len(h.Data) > 10 {
		s += h.Data[:10] + "…\""
	} else {
		s += h.Data + "\""
	}
	s = strings.Replace(s, "\n", `\n`, -1)
	s = strings.Replace(s, "\t", `\t`, -1)
	s = strings.Replace(s, " ", "\u2423", -1)
	return s
}
