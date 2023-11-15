package style

import (
	"golang.org/x/net/html"
)

// Values "default" have the following semantics:
// Treat this as an inherent UA default, which should not be instantiated in memory,
// but rather will be treated implicitely by rendering code.
// See issure https://github.com/npillmayer/tyse/issues/8
//
var nonInherited = map[string]string{
	"position":            "static",
	"background-color":    "default",
	"border-top-color":    "default",
	"border-left-color":   "default",
	"border-right-color":  "default",
	"border-bottom-color": "default",
	"flow-from":           "none",
	"flow-into":           "none",
}

var isDimension = map[string]string{
	"width":                      "auto",
	"height":                     "auto",
	"min-width":                  "none",
	"min-height":                 "none",
	"max-width":                  "none",
	"max-height":                 "none",
	"top":                        "0",
	"right":                      "0",
	"bottom":                     "0",
	"left":                       "0",
	"margin-top":                 "0",
	"margin-left":                "0",
	"margin-right":               "0",
	"margin-bottom":              "0",
	"padding-top":                "0",
	"padding-left":               "0",
	"padding-right":              "0",
	"padding-bottom":             "0",
	"border-top-width":           "medium",
	"border-left-width":          "medium",
	"border-right-width":         "medium",
	"border-bottom-width":        "medium",
	"border-top-left-radius":     "0",
	"border-top-right-radius":    "0",
	"border-bottom-left-radius":  "0",
	"border-bottom-right-radius": "0",
}

// GetUserAgentDefaultProperty returns the user-agent default property for a given key.
func GetUserAgentDefaultProperty(node *html.Node, key string) Property {
	p := NullStyle
	switch key {
	case "display":
		p = DisplayPropertyForHTMLNode(node)
	default:
		if dim, ok := isDimension[key]; ok {
			return Property(dim)
		}
		if p, ok := nonInherited[key]; ok {
			return Property(p)
		}
	}
	// TODO get from user agent defaults,
	//      but how do we access the defaults? as they are attached
	//      to the document root, which we do not have access to from here
	// IDEA We "only" have to include non-inherited properties here
	//      how many are these?
	// Q    should we really always return the default? or rather "" in
	//      cases like color or font, where the layouter will figure out
	//      what best to do. possibly does not want to allocate mem for
	//      styles if all are default => see frame.StyledBox
	return p
}

// DisplayPropertyForHTMLNode returns the default `display` CSS property for an HTML node.
func DisplayPropertyForHTMLNode(node *html.Node) Property {
	if node == nil {
		return "none"
	}
	if node.Type == html.DocumentNode {
		return "block"
	}
	if node.Type != html.ElementNode {
		tracer().Debugf("cannot get display-property for non-element")
		return "none"
	}
	switch node.Data {
	case "head":
		return "none"
	case "p":
		return "block-inline"
	case "html", "aside", "body", "div", "h1", "h2", "h3",
		"h4", "h5", "h6", "it", "ol", "section",
		"ul":
		return "block"
	case "i", "b", "span", "strong":
		return "inline"
	}
	tracer().Infof("unknown HTML element %s/%d will be set to display: block",
		node.Data, node.Type)
	return "block"
}

// InitializeDefaultPropertyValues creates an internal data structure to
// hold all the default values for CSS properties.
// In real-world browsers these are the user-agent CSS values.
func InitializeDefaultPropertyValues(additionalProps []KeyValue) *PropertyMap {
	m := make(map[string]*PropertyGroup, 15)
	root := NewPropertyGroup("Root")

	x := NewPropertyGroup(PGX) // special group for extension properties
	for _, kv := range additionalProps {
		x.Set(kv.Key, kv.Value)
	}
	m[PGX] = x

	margins := NewPropertyGroup(PGMargins)
	margins.Set("margin-top", "0")
	margins.Set("margin-left", "0")
	margins.Set("margin-right", "0")
	margins.Set("margin-bottom", "0")
	margins.Parent = root
	m[PGMargins] = margins

	padding := NewPropertyGroup(PGPadding)
	padding.Set("padding-top", "0")
	padding.Set("padding-left", "0")
	padding.Set("padding-right", "0")
	padding.Set("padding-bottom", "0")
	padding.Parent = root
	m[PGPadding] = padding

	border := NewPropertyGroup(PGBorder)
	border.Set("border-top-color", "black")
	border.Set("border-left-color", "black")
	border.Set("border-right-color", "black")
	border.Set("border-bottom-color", "black")
	border.Set("border-top-width", "medium")
	border.Set("border-left-width", "medium")
	border.Set("border-right-width", "medium")
	border.Set("border-bottom-width", "medium")
	border.Set("border-top-style", "none")
	border.Set("border-left-style", "none")
	border.Set("border-right-style", "none")
	border.Set("border-bottom-style", "none")
	border.Set("border-top-left-radius", "0")
	border.Set("border-top-right-radius", "0")
	border.Set("border-bottom-left-radius", "0")
	border.Set("border-bottom-right-radius", "0")
	border.Parent = root
	m[PGBorder] = border

	dimension := NewPropertyGroup(PGDimension)
	dimension.Set("width", "auto")
	dimension.Set("height", "auto")
	dimension.Set("min-width", "none")
	dimension.Set("min-height", "none")
	dimension.Set("max-width", "none")
	dimension.Set("max-height", "none")
	dimension.Parent = root
	m[PGDimension] = dimension

	region := NewPropertyGroup(PGRegion)
	region.Set("flow-from", "")
	region.Set("flow-into", "")
	region.Parent = root
	m[PGRegion] = region

	display := NewPropertyGroup(PGDisplay)
	display.Set("display", "block")
	display.Set("float", "none")
	display.Set("visibility", "visible")
	display.Set("position", "static")
	display.Parent = root
	m[PGDisplay] = display

	color := NewPropertyGroup(PGColor)
	color.Set("color", "default")
	color.Set("background-color", "default") // TODO set to transparent (CSS default) ?
	color.Parent = root
	m[PGColor] = color

	text := NewPropertyGroup(PGText)
	text.Set("direction", "ltr")
	text.Set("white-space", "normal")
	text.Set("word-spacing", "normal")
	text.Set("letter-spacing", "normal")
	text.Set("word-break", "normal")
	text.Set("overflow-wrap", "normal")
	text.Set("hyphens", "manual")
	text.Parent = root
	m[PGText] = text

	/*
	   type DisplayStyle struct {
	   	Display    uint8 // https://www.tutorialrepublic.com/css-reference/css-display-property.php
	   	Top        dimen.Dimen
	   	Left       dimen.Dimen
	   	Right      dimen.Dimen
	   	Bottom     dimen.Dimen
	   	ZIndex     int
	   	Overflow   uint8
	   	OverflowX  uint8
	   	OverflowY  uint8
	   	Clip       string // geometric shape
	   }

	   type ColorModel string

	   type Color struct {
	   	Color   color.Color
	   	Model   ColorModel
	   	Opacity uint8
	   }

	   type Background struct {
	   	Color color.Color
	   	//Position TODO
	   	Image  image.Image
	   	Origin dimen.Point
	   	Size   dimen.Point
	   	Clip   uint8
	   }

	   type Font struct {
	   	Family     string
	   	Style      string
	   	Variant    uint16
	   	Stretch    uint8
	   	Size       dimen.Dimen
	   	SizeAdjust dimen.Dimen
	   }

	   type TextProperties struct {
	   	Direction          uint8
	   	WordSpacing        uint8
	   	LetterSpacing      uint8
	   	VerticalAlignment  uint8
	   	TextAlignment      uint8 // + TextJustify
	   	TextAlignLast      uint8
	   	TextIndentation    dimen.Dimen // first line
	   	TabSize            dimen.Dimen
	   	LineHeight         uint8
	   	TextDecoration     uint8
	   	TextTransformation uint8
	   	WordWrap           uint8
	   	WordBreak          uint8
	   	Whitespace         uint8
	   	TextOverflow       uint8
	   }


	   type GeneratedContent struct {
	   	Content          string
	   	Quotes           string
	   	CounterReset     uint8
	   	CounterIncrement uint8
	   }

	   type Print struct {
	   	PageBreakAfter  uint8
	   	PageBreakBefore uint8
	   	PageBreakInside uint8
	   }

	   type Outline struct {
	   	Color  color.Color
	   	Offset dimen.Dimen
	   	Style  uint8
	   	Width  dimen.Dimen
	   }

	   //list-style-type:
	   //	disc | circle | square | decimal | decimal-leading-zero | lower-roman |
	   //  upper-roman | lower-greek | lower-latin | upper-latin | armenian |
	   //  georgian | lower-alpha | upper-alpha | none | initial | inherit
	   type List struct {
	   	StyleImage    image.Image
	   	StylePosition uint8 // inside, outside
	   	StyleType     uint8
	   }

	*/

	return &PropertyMap{m}
}
