/*
Package ot provides access to OpenType font tables and features.
Intended audience for this package are:

▪︎ text shapers, such as HarfBuzz (https://harfbuzz.github.io/what-does-harfbuzz-do.html)

▪︎ glyph rasterizers, such as FreeType (https://github.com/golang/freetype)

▪︎ any application needing to have the internal structure of an OpenType font file available,
and possibly extending the methods of package `ot` by handling additional font tables

Package `ot` will not provide functions to interpret any table of a font, but rather
just expose the tables to the client. For example, it is not possible to ask
package `ot` for a kerning distance between two glyphs. Clients have to check
for the availability of kerning information and consult the appropriate table(s)
themselves. From this point of view, `ot` is a low-level package.
Functions for getting kerning values and other layout directives from a font
are homed in a sister package.
Likewise, this package is not intended for font manipulation applications (you may check out
https://pkg.go.dev/github.com/ConradIrwin/font for that).

OpenType fonts contain a whole lot of different tables and sub-tables. This package
strives to make the semantics of the tables accessible, thus has a lot of different
types for the different kinds of OT tables. This makes `ot` a shallow API,
but it will nevertheless abstract away some implementation details of fonts:

▪︎ Format versions: many OT tables may occur in a variety of formats. Tables in `ot` will
hide the concrete format and structure of underlying OT tables.

▪︎ Word size: offsets in OT may either be 2-byte or 4-byte values. Package `ot` will
hide offset-related details (see section below).

▪︎ Bugs in fonts: many fonts in the wild contain entries that—strictly speaking—infringe
upon the OT specification (for example, Calibri has an overflow in a 'kern' table variable),
but an application using it should not fail because of recoverable errors.
Package `ot` will try to circumvent known bugs in common fonts.

# Schrödinger's Cat

OpenType fonts contain quite a multitude of tables, and a package intended to
expose the semantics of OT tables ought to wrap each of these into a Go type.
However, this would waste both time and space, as the usage of subsets of these
tables is mutually exclusive (think 'head' and 'bhea'). And it would make the API
of package `ot` even more broad as it already is. We therefore focus on the most
important tables to be semantically wrapped in Go types and find a different
approach for other tables and their OT data structures.

The binary data of a font can be thought of as a bunch of structures
connected by links. The linking is done by offsets (u16 or u32) from link anchors
defined by the spec. Data-structures may be categorized into fields-like,
list-like and map-like. The implementation details of these structures vary
heavily, and many internal tables combine more than one category, but conceptually it
should be possible to navigate the graph, spanned by links and structures,
without caring about implementation details.
This is where the cat comes in.

We design an abstraction resting on chains of navigation items and links. There
surely is a fancy name from functional theory on this (monads on functions on font data),
but I prefer to think about them as Schrödinger's cat: In the end you have to
open the box in order to know if the cat is alive or dead. Before we get too
quantum, however, let's consider an example. The OpenType specification flags the
'OS/2' table as mandatory (though unused on Mac platforms), but package `ot` does
not offer a type for it. As a client, how do you access, e.g., OS/2.xAvgCharWidth?
We start by requesting OS/2 as a vanilla table:

	os2 := myfont.Table[T("OS/2")]

This should succeed unless `myfont` is broken. From there on, clients will have
to consult the OpenType specification. That will tell them that xAvgCharWidth is
the 2nd field (index 1) of table OS/2, right after the version field.

	xAvgCharWidth := os2.Fields().Get(1).U16(0)

This will read xAvgCharWidth as an uint16, which is the data type of xAvgCharWidth
according to the spec. That sure looks like a complicated way of getting a number
out of a struct, but remember that we agreed on having an abstraction on top of
field-likes, list-likes and map-likes. You didn't have to think about version
differences in OT OS/2-tables or byte-offsets from anchors.

But I promised you a cat, you say? In fact, the example already had one included,
but's let's head over to a bigger cat. The most complex OT tables, apart from
glyphs themselves, include the so-called “layout-tables”.

	calibri := ot.Parse(…)                     // find font 'Calibri' on our system
	gsub := calibri.Table(T("GSUB")).Self().AsGSub()            // semantic Go type

GSUB is an important table for text shaping, so package `ot` offers a special type.
However, this type is not exposing GSUB in full depth! Thus we type:

	feats := gsub.ScriptList.LookupTag(T("latn")).Navigate().Map().LookupTag(T("TRK")).Navigate().List()
	fmt.Println("%d features for Turkish", feats.Len())
	// => yields 24

If you happen to not know the OpenType specification by heart, I'll help you out:
We want to know if the font contains features applicable for Latin script with
Turkish language flavour ('Calibri' actually does).
The line of code is quite a mouthful, I'll readily concede. However, if you ever wrote
code to extract information from a font file, you'd probably be used to a lot of error
branches like this:

	latinScript, ok := GSUB.lookup("latn")                 // pseudo code
	if !ok {
	    … // opt out
	}
	turkishLangRecord, ok := latinScript.lookup("TRK")     // pseudo code
	if !ok {
	    … // opt out (use default)
	}
	// etc …

So in effect, we're checking multiple times if the cat is still okay, until we
eventually reach the box we're looking for. Package `ot` will let you travel the
distance without reboxing the cat and just check once at the end. In other
words, it's null-safe (imagine some kind of hidden Maybe-type).

Some problems arise with such an approach:

▪︎ Neither syntax nor type-system do guide you, but clients rather have to consult the
OpenType specification (not a fun read).

▪︎ Oftentimes you need some trial-and-error to find the right navigation items.
It's hard to completely document every path, because then one would re-write the
OT tables' spec.

▪︎ To fit every OT-structure in the mould of the aforementioned three abstractions,
some OT tables have to be “bended”. It's sometimes not intuitively clear what
abstraction is the best, or which one `ot` would choose.

At the end of the day we will see if this approach does the job when we gain
some experience with using it from a client perspective.

# Status

Work in progress. Handling fonts is fiddly and fonts have become complex software
applications in their own right. I often need a break from the vast desert of
bytes (without any sign posts), which is what font data files are at their core. A break
where I talk to myself and ask, this is what you do in your spare time? Really?

No font collections nor variable fonts are supported yet, but will be in time.

# License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © Norbert Pillmayer <norbert@pillmayer.com>

Some code has originally been copied over from golang.org/x/image/font/sfnt/cmap.go,
as the cmap-routines are not accessible through the sfnt package's API.
I understand this to be legally okay as long as the Go license information
stays intact.

	Copyright 2017 The Go Authors. All rights reserved.
	Use of this source code is governed by a BSD-style
	license that can be found in the LICENSE file.

The license file mentioned can be found in file GO-LICENSE at the root folder
of this module.
*/
package ot

/*
There are (at least) two Go packages around for parsing SFNT fonts:

▪ https://pkg.go.dev/golang.org/x/image/font/sfnt

▪ https://pkg.go.dev/github.com/ConradIrwin/font/sfnt

It's always a good idea to prefer packages from the Go core team, and the
x/image/font/sfnt package is certainly well suited for rasterizing applications
(as proven by the test cases). However, it is less well suited as a basis for
the task of text-shaping. This task requires access to the tables contained in
a font and means of navigating them, cross-checking entries, applying different
shaping algorithms, etc. Moreover, the API is not intended to be extended by
other packages, but has been programmed with a concrete target in mind.

ConradIrwin/font allows access to the font tables it has parsed. However, its
focus is on font file manipulation (read in ⇒ manipulate ⇒ export), thus
access to tables means more or less access to the tables binaries and
doing much of the interpretation on the client side. I started out pursuing this
approach, but at the end abondened it. The main reason for this is that I
prefer the approach of the Go core team of keeping the initial font binary
in memory, and not copying out too much into separate buffers or data structures.
I need to have the binary data in memory anyway, as for complex-script shaping
we will rely on HarfBuzz for a long time to come (HarfBuzz receives a font
as a byte-blob and does its own font parsing).

A better suited blueprint of what we're trying to accomplish is this implementation
in Rust:

▪︎ https://github.com/bodoni/opentype

*/

import (
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
)

// Valuable resource:
// http://opentypecookbook.com/

// tracer writes to trace with key 'tyse.fonts'
func tracer() tracing.Trace {
	return tracing.Select("tyse.fonts")
}

// errFontFormat produces user level errors for font parsing.
func errFontFormat(x string) error {
	return core.Error(core.EINVALID, "OpenType font format: %s", x)
}
