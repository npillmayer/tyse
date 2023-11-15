/*
Package font is for typeface and font handling.

There is a certain confusion with the nomenclature of typesetting. We will
stick to the following definitions:

▪︎ A "typeface" is a family of fonts. An example is "Helvetica".
This corresponds to a TrueType "collection" (*.ttc).

▪︎ A "scalable font" is a font, i.e. a variant of a typeface with a
certain weight, slant, etc.  An example is "Helvetica regular".

▪︎ A "typecase" is a scaled font, i.e. a font in a certain size for
a certain script and language. The name is reminiscend on the wooden
boxes of typesetters in the era of metal type.
An example is "Helvetica regular 11pt, Latin, en_US".

Please note that Go (Golang) does use the terms "font" and "face"
differently–actually more or less in an opposite manner.

# Status

Does not yet contain methods for font collections (*.ttc), e.g.,
/System/Library/Fonts/Helvetica.ttc on Mac OS.

# Links

OpenType explained:
https://docs.microsoft.com/en-us/typography/opentype/

______________________________________________________________________

# License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>
*/
package font

import (
	"fmt"
	"os"
	"sync"

	"github.com/npillmayer/schuko/tracing"
	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// tracer writes to trace with key 'tyse.font'
func tracer() tracing.Trace {
	return tracing.Select("tyse.font")
}

const (
	StyleNormal = xfont.StyleNormal
	StyleItalic = xfont.StyleItalic
)

const (
	WeightLight    = xfont.WeightLight
	WeightNormal   = xfont.WeightNormal
	WeightSemiBold = xfont.WeightSemiBold
	WeightBold     = xfont.WeightBold
)

// ScalableFont is an internal representation of an outline-font of type
// TTF of OTF.
type ScalableFont struct {
	Fontname string
	Filepath string     // file path
	Binary   []byte     // raw data
	SFNT     *sfnt.Font // the font's container // TODO: not threadsafe???
}

// TypeCase represents a font at a specific point size, e.g. "Helvetica bold 10pt".
type TypeCase struct {
	scalableFontParent *ScalableFont
	font               xfont.Face // Go uses 'face' and 'font' in an inverse manner
	size               float32
	// script
	// language
}

func NullTypeCase() *TypeCase {
	return &TypeCase{
		font: nil,
		size: 10,
	}
}

// LoadOpenTypeFont loads an OpenType font (TTF or OTF) from a file.
func LoadOpenTypeFont(fontfile string) (*ScalableFont, error) {
	bytez, err := os.ReadFile(fontfile)
	if err != nil {
		return nil, err
	}
	return ParseOpenTypeFont(bytez)
}

// ParseOpenTypeFont loads an OpenType font (TTF or OTF) from memory.
func ParseOpenTypeFont(fbytes []byte) (f *ScalableFont, err error) {
	f = &ScalableFont{Binary: fbytes}
	f.SFNT, err = sfnt.Parse(f.Binary)
	if err != nil {
		return nil, err
	}
	if f.Fontname, err = f.SFNT.Name(nil, sfnt.NameIDFull); err == nil {
		tracer().Debugf("loaded and parsed SFNT %s", f.Fontname)
	}
	return
}

// PrepareCase prepares a typecase in a given point size, e.g. "Helvetica bold 10pt"
// from an existing font "Helvetiva bold", which has been previously loaded.
func (sf *ScalableFont) PrepareCase(fontsize float32) (*TypeCase, error) {
	// TODO: check if language fits to script
	// TODO: check if font supports script
	typecase := &TypeCase{}
	typecase.scalableFontParent = sf
	if fontsize < 5.0 || fontsize > 500.0 {
		fmt.Printf("prepare typecase: size must be 5pt < size < 500pt, is %g (set to 10pt)\n", fontsize)
		fontsize = 10.0
	}
	options := &opentype.FaceOptions{
		Size: float64(fontsize),
		DPI:  600,
	}
	f, err := opentype.NewFace(sf.SFNT, options)
	if err == nil {
		typecase.font = f
		typecase.size = fontsize
	}
	return typecase, err
}

// ScalableFontParent returns the unscaled font a typecase has been derived from.
func (tc *TypeCase) ScalableFontParent() *ScalableFont {
	return tc.scalableFontParent
}

// PtSize returns the point-size of a typecase.
func (tc *TypeCase) PtSize() float32 {
	return tc.size
}

// Metrics returns a font's metrics.
func (tc *TypeCase) Metrics() xfont.Metrics {
	return tc.font.Metrics()
}

// --- Fallback font ---------------------------------------------------------

// FallbackFont returns a font to be used if everything else failes. It is
// always present. Currently we use Go Sans.
func FallbackFont() *ScalableFont {
	fallbackFontLoading.Do(func() {
		fallbackFont = loadFallbackFont()
	})
	return fallbackFont
}

var fallbackFontLoading sync.Once

// fallbackFont is a font that is used if everything else failes.
// Currently we use Go Sans.
var fallbackFont *ScalableFont

func loadFallbackFont() *ScalableFont {
	var err error
	gofont := &ScalableFont{
		Fontname: "Go Sans",
		Filepath: "internal",
		Binary:   goregular.TTF,
	}
	gofont.SFNT, err = sfnt.Parse(gofont.Binary)
	if err != nil {
		panic("cannot load default font") // this cannot happen
	}
	return gofont
}

// ---------------------------------------------------------------------------

// TODO make this something like Apple's font descriptors

// Descriptor represents all the known variants of a font.
type Descriptor struct {
	Family   string   `json:"family"`
	Variants []string `json:"variants"`
	Path     string   // only used if just a single variant
}

// ---------------------------------------------------------------------------

/*
u/em   = 2000
_em    = 12 pt  = 0,1666 in
_dpi   = 120
=>
_d/_em = 120 * 0,1666 = 19,992 pixels per em
=>
u1     = 150

2000 = 19,992
u1   = ?
=>  ? = _d/em * u1 / u/em

_u1    = 150 / _d/_em  = 7,503  pixels

Beispiel:
PT  = 12
DPI = 72
_d/_em = gtx.Px(DPI) * (PT / 72.27)
=> gtx.Px(12)  vereinfacht bei dpi = 72
*/

// PtIn is 72.27, i.e. printer's points per inch.
var PtIn fixed.Int26_6 = fixed.I(72) + fixed.I(27)/100

// PpEm calculates a ppem value for a given font point-size and an output resolution (dpi).
func PpEm(ptSize fixed.Int26_6, dpi float32) fixed.Int26_6 {
	_dpi := fixed.Int26_6(dpi * 64)
	return _dpi * (ptSize / PtIn)
}

// RasterCoords transforms `u`, a value in font-units, into pixel coordinates.
// Calculation is done for a font `sfont` at a given point-size `ptSize`.
func RasterCoords(u sfnt.Units, sfont *sfnt.Font, ptSize fixed.Int26_6, dpi float32) fixed.Int26_6 {
	_ppem := PpEm(ptSize, dpi)
	uem := sfont.UnitsPerEm()
	_uem := fixed.I(int(uem))
	_u := fixed.I(int(u)) * _ppem / _uem
	return _u
}
