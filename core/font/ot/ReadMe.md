
ot – OpenType Font Tables and Features
---------------------------------------

Package `ot` provides access to OpenType font tables and features.
The intended audience for this package are:

*︎ text shapers, such as HarfBuzz (https://harfbuzz.github.io/what-does-harfbuzz-do.html)

*︎ glyph rasterizers, such as FreeType (https://github.com/golang/freetype)

*︎ any application needing to have the internal structure of an OpenType font file available, and possibly extending the methods of this module by handling additional font tables 

This package is *not* intended for font manipulation applications. You may check out
https://pkg.go.dev/github.com/ConradIrwin/font
for this.

### Status

This is very much work in progress.
Handling fonts is fiddly and fonts have become complex software
applications in their own right. I often need a break from the vast desert of
bytes (without any sign posts), which is what font data files are at their core. Breaks,
where I talk to myself and ask, this is what you do in your spare time? Really?

### Other Solutions to Font Parsing

There are (at least) two Go packages around for parsing SFNT fonts:

* https://pkg.go.dev/golang.org/x/image/font/sfnt

* https://pkg.go.dev/github.com/ConradIrwin/font/sfnt

It’s always a good idea to prefer packages from the Go core team, and the
x/image/font/sfnt package is certainly well suited for rasterinzing applications
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

* https://github.com/bodoni/opentype
