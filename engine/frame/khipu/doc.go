// Package khipu is about encoding text into typesetting items.
//
// "Khipu were recording devices fashioned from strings
// historically used by a number of cultures in the region of
// Andean South America.
// Khipu is the word for "knot" in Cusco Quechua.
// A khipu usually consisted of cotton or camelid fiber strings. The Inca
// people used them for collecting data and keeping records, monitoring tax
// obligations, properly collecting census records, calendrical information,
// and for military organization. The cords stored numeric and other values
// encoded as knots, often in a base ten positional system. A khipu could
// have only a few or thousands of cords."
// ––Excerpt from a Wikipedia article about khipus
//
// The Khipukamayuqs (Quechua for “knot-makers”) were the scribes of those
// times, tasked with encoding tax figures and other administrative
// information in knots.
// We will use this analogy to call typesetting items "khipus" or "knots",
// and objects which produce khipus will be "Khipukamayuqs".
// Knots implement items for typesetting paragraphs. We will use a
// box-and-glue model, the various knot types more or less implementing
// the corresponding node types from the TeX typesetting system.
//
// A Khipukamayuqs is part of a typsetting pipeline and will transform
// text into khipus.
// Khipus are the input for linebreakers. The overall process of creating
// them and the interaction with line breaking looks like this:
//
// Create Khipus from Text
//
// (1) Normalize Unicode text
//
// 	https://godoc.org/golang.org/x/text/unicode/norm
//
// (2) Find natural text wrap opportunities
// (words in many scripts, syllables/character in East Asia, etc.)
//
// 	https://godoc.org/github.com/npillmayer/uax
//
// (3) Bi-directional text
//
// 	https://godoc.org/github.com/npillmayer/uax/bidi
// 	https://godoc.org/golang.org/x/text/unicode/bidi
// 	https://www.w3.org/International/articles/inline-bidi-markup/
//
// (4) Hyphenation:
// Lliang patterns + language-specific code
//
// 	https://godoc.org/github.com/npillmayer/tyse/core/hyphenation
//
// (5) Translate feasible breakpoints to penalties, glue and discretionaries
//
//     https://wiki.apache.org/xmlgraphics-fop/KnuthsModel
//
// (6) Shape text -> Glyphs
// + alternative glyphs (end-of-line condensed in Arabic, etc.)
//
// 	http://behdad.org/text/
//
// At this point, text has been fully converted to khipus.
//
/*
License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package khipu

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'tyse.khipu'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.khipu")
}
