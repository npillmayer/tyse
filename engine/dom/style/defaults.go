package style

import (
	"github.com/npillmayer/schuko/gtrace"
	"golang.org/x/net/html"
)

var nonInherited = map[string]bool{
	"display":   true,
	"position":  true,
	"flow-from": true,
	"flow-into": true,
}

// GetDefaultProperty returns the default property for a given key.
func GetDefaultProperty(styler Styler, key string) Property {
	p := NullStyle
	switch key {
	case "display":
		p = DisplayPropertyForHTMLNode(styler.HTMLNode())
	}
	// TODO get from user agent defaults
	return p
}

// DisplayPropertyForHTMLNode returns the *display* CSS property for an HTML node.
func DisplayPropertyForHTMLNode(node *html.Node) Property {
	if node == nil {
		return "none"
	}
	if node.Type == html.DocumentNode {
		return "block"
	}
	if node.Type != html.ElementNode {
		T().Debugf("cannot get display-property for non-element")
		return "none"
	}
	switch node.Data {
	case "html", "aside", "body", "div", "h1", "h2", "h3",
		"h4", "h5", "h6", "it", "ol", "p", "section",
		"ul":
		return "block"
	case "i", "b", "span", "strong":
		return "inline"
	}
	gtrace.EngineTracer.Infof("unknown HTML element %s/%d will be set to display: block",
		node.Data, node.Type)
	return "block"
}
