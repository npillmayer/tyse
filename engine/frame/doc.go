/*
Package frame deals with typesetting frames.

Typesetting may be understood as the process of placing boxes within
larger boxes. The smallest type of box is a glyph, i.e. a printable
letter. The largest type of box is a page—or even a book, where
page-boxes are placed into.

The box model is very versatile. Nevertheless we will generalize the
notion of a box to mean the bounding box of a polygon. Typesetting in
irregular shapes is a feature available in most modern systems, e.g.
when letting text flow around a non-rectangular illustration.

This module deals with rectangular boxes, starting at the glyph level.
Boxes follow the CSS box model. Nevertheless, the notation oftentimes follows
the one introduced by the TeX typesetting system.

______________________________________________________________________

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package frame

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'tyse.frame'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.frame")
}
