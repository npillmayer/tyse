/*
Package otshaper is about OpenType text shaping (work in progress).

From the Harfbuzz documentation (https://harfbuzz.github.io/what-is-harfbuzz.html):

“Text shaping is the process of translating a string of character codes (such
as Unicode codepoints) into a properly arranged sequence of glyphs that can be
rendered onto a screen or into final output form for inclusion in a document.
The shaping process is dependent on the input string, the active font, the script
(or writing system) that the string is in, and the language that the string is in.”

For a thorough introduction take a look at this document:
https://github.com/n8willis/opentype-shaping-documents/tree/master.

From the OpenType spec
(https://docs.microsoft.com/en-us/typography/opentype/spec/chapter2#features-and-lookups):

OpenType Layout features and lookups define information that is specific to the glyphs in a given font.
They do not encode information that is constant within the conventions of a particular language or the
typography of a particular script. Information that would be replicated across all fonts in a given
language belongs in the text-processing application for that language, not in the fonts.

# License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © Norbert Pillmayer <norbert@pillmayer.com>
*/
package otshaper

import (
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
	"github.com/npillmayer/tyse/core/font/opentype/ot"
)

// NODEF represents OpenType `.notdef`.
const NOTDEF = ot.GlyphIndex(0)

// tracer writes to trace with key 'tyse.fonts'
func tracer() tracing.Trace {
	return tracing.Select("tyse.fonts")
}

// errShaper produces user level errors for text shaping.
func errShaper(x string) error {
	return core.Error(core.EINVALID, "OpenType text shaping: %s", x)
}

// assert emulates assertions known from other programming languages.
// Dealing with fonts and Unicode text sometimes feels quite brittle: a lot of things
// can go wrong with uncertainties with coding, errors in fonts, etc.
// Until things work out to be robust enough, we'll be defensive.
func assert(condition bool, msg string) {
	if !condition {
		panic(msg)
	}
}
