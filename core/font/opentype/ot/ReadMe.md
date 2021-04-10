
ot – OpenType Font Tables and Features
---------------------------------------

Package `ot` provides access to OpenType font tables and features.
The intended audience for this package are:

*︎ text shapers, such as HarfBuzz (<https://harfbuzz.github.io/what-does-harfbuzz-do.html>)

*︎ glyph rasterizers, such as FreeType (<https://github.com/golang/freetype>)

*︎ any application needing to have the internal structure of an OpenType font file available, and possibly extending the methods of this module by handling additional font tables 

This package is *not* intended for font manipulation applications. You may check out
<https://pkg.go.dev/github.com/ConradIrwin/font>
for this.

### Status

This is very much work in progress.
Handling fonts is fiddly and fonts have become complex software
applications in their own right. I often need a break from the vast desert of
bytes (without any sign posts), which is what font data files are at their core. Breaks,
where I talk to myself and ask, this is what you do in your spare time? Really?

 No font collections nor variable fonts are supported yet. 

### Other Solutions to Font Parsing

There are (at least) two Go packages around for parsing SFNT fonts:

* <https://pkg.go.dev/golang.org/x/image/font/sfnt>
* <https://pkg.go.dev/github.com/ConradIrwin/font/sfnt>

It’s always a good idea to prefer packages from the Go core team, and the
x/image/font/sfnt package is certainly well suited for rasterizing applications
(as proven by the test cases). However, it is less well suited as a basis for
the task of text-shaping. That task requires access to the tables contained in
a font and utilities for navigating them, cross-checking entries, applying different
shaping algorithms, etc. Moreover, the API is not intended to be extended by
other packages, but has been programmed with a concrete target in mind.

ConradIrwin/font allows access to the font tables it has parsed. However, its
focus is on font file manipulation (read in ⇒ manipulate ⇒ export), thus
access to tables means more or less access to the tables binaries and
doing much of the interpretation on the client side. I started out pursuing this
approach, but at the end abondened it. The main reason is that I
prefer the approach of the Go core team of keeping the initial font binary
in memory, and not copying out too much into separate buffers or data structures.
I need to have the binary data in memory anyway, as for complex-script shaping
we will rely on HarfBuzz for a long time to come (HarfBuzz receives a font
as a byte-blob and does its own font parsing).

A better suited blueprint of what we're trying to accomplish is this implementation
in Rust:

* https://github.com/bodoni/opentype

### Abstractions

Package `ot` will not provide functions to interpret any table of a font, but rather
expose the tables to the client in a semantic way. It is not possible to, for example, ask
package `ot` for a kerning distance between two glyphs. Instead, clients have to check
for the availability of kerning tables and consult the appropriate table(s)
themselves. From this point of view `ot` is a low-level package.

The binary data of a font can be thought of as a bunch of structures
connected by links. The linking is done by offsets (u16 or u32) from link anchors
defined by the spec. Data-structures may be categorized into fields-like,
list-like and map-like. The implementation details of these structures vary
heavily, and many internal tables combine more than one category, but conceptually it
should be possible to navigate the graph, spanned by links and structures,
without caring about implementation details.
Consider this overview of OpenType Layout tables (GSUB and GPOS):

<div style="width:580px;padding:5px;padding-bottom:10px">
<img alt="OpenType structure for layout tables"
 src="http://npillmayer.github.io/img/OpenType-layout-table.svg"
 width="580px">
</div>

GSUB is an important table for text shaping, so package `ot` offers a special semantic type.
However, this type is not exposing GSUB in full depth.
To find out if the current font contains features applicable for Latin script with
Turkish language flavour, type:

    langSys := gsub.ScriptList.LookupTag(T("latn")).Navigate().Map().LookupTag(T("TRK")).Navigate().List()
    fmt.Println("%d font-features for Turkish", langSys.Len())
    // => yields 24 with font 'Calibri'

This is an early draft, not suited to be used by other programs.
