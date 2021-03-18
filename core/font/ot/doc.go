/*
Package ot provides access to OpenType font tables and features.
The intended audience for this package are:

▪︎ text shapers, such as HarfBuzz (https://harfbuzz.github.io/what-does-harfbuzz-do.html)

▪︎ glyph rasterizers, such as FreeType (https://github.com/golang/freetype)

▪︎ any application needing to have the internal structure of an OpenType font file available,
and possibly extending the methods of this module by handling additional font tables

This package is *not* intended for font manipulation applications. You may check out
https://pkg.go.dev/github.com/ConradIrwin/font
for this.

Status

Work in progress. Handling fonts is fiddly and fonts have become complex software
applications in their own right. I often need a break from the vast desert of
bytes (without any sign posts), which is what font data files are at their core. Breaks,
where I talk to myself and ask, this is what you do in your spare time? Really?

License

Some code has originally been copied over from golang.org/x/image/font/sfnt/cmap.go,
as the cmap-routines are not accessible through the sfnt package's API.
I understand this to be legally okay as long as the Go license information
stays intact.

    Copyright 2017 The Go Authors. All rights reserved.
    Use of this source code is governed by a BSD-style
    license that can be found in the LICENSE file.

The license file mentioned can be found in file GO-LICENSE at the root folder
of this module.

Code in this module beyond the x/image/font/sfnt parts is subject to the following license:

BSD 3-Clause License

Copyright (c) 2020–21, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
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
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
)

// trace traces to a global core-tracer.
func trace() tracing.Trace {
	return gtrace.CoreTracer
}

// errFontFormat produces user level errors for font parsing.
func errFontFormat(x string) error {
	return core.Error(core.EINVALID, "OpenType font format: %s", x)
}
