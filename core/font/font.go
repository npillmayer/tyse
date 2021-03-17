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

Status

Does not yet contain methods for font collections (*.ttc), e.g.,
/System/Library/Fonts/Helvetica.ttc on Mac OS.

Links

OpenType explained:
https://docs.microsoft.com/en-us/typography/opentype/

----------------------------------------------------------------------

BSD License

Copyright (c) 2017-21, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE. */
package font

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"golang.org/x/image/font"
	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// trace traces to a global core-tracer.
func trace() tracing.Trace {
	return gtrace.CoreTracer
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

//TypeCase represents a font at a specific point size, e.g. "Helvetica bold 10pt".
type TypeCase struct {
	scalableFontParent *ScalableFont
	font               font.Face // Go uses 'face' and 'font' in an inverse manner
	size               float64
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
	bytez, err := ioutil.ReadFile(fontfile)
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
	f.Fontname, _ = f.SFNT.Name(nil, sfnt.NameIDFull)
	return
}

// PrepareCase prepares a typecase in a given point size, e.g. "Helvetica bold 10pt"
// from an existing font "Helvetiva bold", which has been previously loaded.
func (sf *ScalableFont) PrepareCase(fontsize float64) (*TypeCase, error) {
	// TODO: check if language fits to script
	// TODO: check if font supports script
	typecase := &TypeCase{}
	typecase.scalableFontParent = sf
	if fontsize < 5.0 || fontsize > 500.0 {
		fmt.Printf("prepare typecase: size must be 5pt < size < 500pt, is %g (set to 10pt)\n", fontsize)
		fontsize = 10.0
	}
	options := &opentype.FaceOptions{
		Size: fontsize,
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
func (tc *TypeCase) PtSize() float64 {
	return tc.size
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

// --- Font Registry ---------------------------------------------------------

// Registry is a type for holding information about loaded fonts for a
// typesetter.
type Registry struct {
	sync.Mutex
	fonts     map[string]*ScalableFont
	typecases map[string]*TypeCase
}

var globalFontRegistry *Registry

var globalRegistryCreation sync.Once

// GlobalRegistry is an application-wide singleton to hold information about
// loaded fonts and typecases.
func GlobalRegistry() *Registry {
	globalRegistryCreation.Do(func() {
		globalFontRegistry = NewRegistry()
	})
	return globalFontRegistry
}

func NewRegistry() *Registry {
	fr := &Registry{
		fonts:     make(map[string]*ScalableFont),
		typecases: make(map[string]*TypeCase),
	}
	return fr
}

// StoreFont pushes a font into the registry if it isn't contained yet.
//
// The font will be stored using the normalized font name as a key. If this
// key is already associated with a font, that font will not be overridden.
func (fr *Registry) StoreFont(normalizedName string, f *ScalableFont) {
	if f == nil {
		trace().Errorf("registry cannot store null font")
		return
	}
	fr.Lock()
	defer fr.Unlock()
	//style, weight := GuessStyleAndWeight(f.Fontname)
	//fname := NormalizeFontname(f.Fontname, style, weight)
	if _, ok := fr.fonts[normalizedName]; !ok {
		trace().Debugf("registry stores font %s as %s", f.Fontname, normalizedName)
		fr.fonts[normalizedName] = f
	}
}

// TypeCase returns a concrete typecase with a given font, style, weight and size.
// If a suitable typecase has already been cached, TypeCase will return the cached
// typecase. If a suitable font has previously been stored under key
// `normalizedName`, a typecase will be derived from this font.
//
// If not typecase can be produced, TypeCase will derive one from a system-wide
// fallback font and return it, together with an error message.
//
func (fr *Registry) TypeCase(normalizedName string, size float64) (*TypeCase, error) {
	//
	trace().Debugf("registry searches for font %s at %.2f", normalizedName, size)
	//fname := NormalizeFontname(name, style, weight)
	tname := appendSize(normalizedName, size)
	fr.Lock()
	defer fr.Unlock()
	if t, ok := fr.typecases[tname]; ok {
		trace().Infof("registry found font %s", tname)
		return t, nil
	}
	if f, ok := fr.fonts[normalizedName]; ok {
		t, err := f.PrepareCase(size)
		trace().Infof("font registry has font %s, caches at %.2f", normalizedName, size)
		t.scalableFontParent = f
		fr.typecases[tname] = t
		return t, err
	}
	trace().Infof("registry does not contain font %s", normalizedName)
	err := errors.New("font " + normalizedName + " not found in registry")
	//
	// store typecase from fallback font, if not present yet, and return it
	fname := "fallback"
	tname = appendSize("fallback", size)
	if t, ok := fr.typecases[fname]; ok {
		return t, err
	}
	f := FallbackFont()
	t, _ := f.PrepareCase(size)
	trace().Infof("font registry caches fallback font %s at %.2f", fname, size)
	fr.fonts[fname] = f
	fr.typecases[tname] = t
	return t, err
}

// LogFontList is a helper function to dump the list of known fonts and typecases
// in a registry to the trace-file (log-level Info).
func (fr *Registry) LogFontList() {
	level := trace().GetTraceLevel()
	trace().SetTraceLevel(tracing.LevelInfo)
	trace().Infof("--- registered fonts ---")
	for k, v := range fr.fonts {
		trace().Infof("font [%s] = %v", k, v.Fontname)
	}
	for k, v := range fr.typecases {
		trace().Infof("typecase [%s] = %v", k, v.scalableFontParent.Fontname)
	}
	trace().Infof("------------------------")
	trace().SetTraceLevel(level)
}

func NormalizeFontname(fname string, style xfont.Style, weight xfont.Weight) string {
	fname = strings.TrimSpace(fname)
	fname = strings.ReplaceAll(fname, " ", "_")
	if dot := strings.LastIndex(fname, "."); dot > 0 {
		fname = fname[:dot]
	}
	fname = strings.ToLower(fname)
	switch style {
	case xfont.StyleItalic, xfont.StyleOblique:
		fname += "-italic"
	}
	switch weight {
	case xfont.WeightLight, xfont.WeightExtraLight:
		fname += "-light"
	case xfont.WeightBold, xfont.WeightExtraBold, xfont.WeightSemiBold:
		fname += "-bold"
	}
	return fname
}

func appendSize(fname string, size float64) string {
	fname = fmt.Sprintf("%s-%.2f", fname, size)
	return fname
}

// GuessStyleAndWeight trys to guess a font's style and weight from the
// font's file name.
func GuessStyleAndWeight(fontfilename string) (xfont.Style, xfont.Weight) {
	fontfilename = path.Base(fontfilename)
	ext := path.Ext(fontfilename)
	fontfilename = strings.ToLower(fontfilename[:len(fontfilename)-len(ext)])
	s := strings.Split(fontfilename, "-")
	if len(s) > 1 {
		switch s[len(s)-1] {
		case "light", "xlight":
			return xfont.StyleNormal, xfont.WeightLight
		case "normal", "medium", "regular", "r":
			return xfont.StyleNormal, xfont.WeightNormal
		case "bold", "b":
			return xfont.StyleNormal, xfont.WeightBold
		case "xbold", "black":
			return xfont.StyleNormal, xfont.WeightExtraBold
		}
	}
	style, weight := xfont.StyleNormal, xfont.WeightNormal
	if strings.Contains(fontfilename, "italic") {
		style = xfont.StyleItalic
	}
	if strings.Contains(fontfilename, "light") {
		weight = xfont.WeightLight
	}
	if strings.Contains(fontfilename, "bold") {
		weight = xfont.WeightBold
	}
	return style, weight
}

// Matches returns true if a font's filename contains pattern and indicators
// for a given style and weight.
func Matches(fontfilename, pattern string, style xfont.Style, weight xfont.Weight) bool {
	basename := path.Base(fontfilename)
	basename = basename[:len(basename)-len(path.Ext(basename))]
	basename = strings.ToLower(basename)
	trace().Debugf("basename of font = %s", basename)
	if !strings.Contains(basename, strings.ToLower(pattern)) {
		return false
	}
	s, w := GuessStyleAndWeight(basename)
	if s == style && w == weight {
		return true
	}
	return false
}

// Descriptor represents all the known variants of a font.
type Descriptor struct {
	Family   string   `json:"family"`
	Variants []string `json:"variants"`
	Path     string   // only used if just a single variant
}

// MatchConfidence is a type for expressing the confidence level of font matching.
type MatchConfidence int

const (
	NoConfidence      MatchConfidence = 0
	LowConfidence     MatchConfidence = 2
	HighConfidence    MatchConfidence = 3
	PerfectConfidence MatchConfidence = 4
)

// ClosestMatch scans a list of font desriptors and returns the closest match
// for a given set of parametesrs.
// If no variant matches, returns `NoConfidence`.
//
func ClosestMatch(fdescs []Descriptor, pattern string, style xfont.Style,
	weight xfont.Weight) (match Descriptor, variant string, confidence MatchConfidence) {
	//
	r, err := regexp.Compile(strings.ToLower(pattern))
	if err != nil {
		trace().Errorf("invalid font name pattern")
		return
	}
	for _, fdesc := range fdescs {
		//trace().Debugf("trying to match %s", strings.ToLower(fdesc.Family))
		if !r.MatchString(strings.ToLower(fdesc.Family)) {
			continue
		}
		for _, v := range fdesc.Variants {
			s := MatchStyle(v, style)
			w := MatchWeight(v, weight)
			if (s+w)/2 > confidence {
				//trace().Debugf("variant %+v match confidence = %d + %d", v, s, w)
				confidence = (s + w) / 2
				variant = v
				match = fdesc
			}
		}
	}
	return
}

// ---------------------------------------------------------------------------

// MatchStyle trys to match a font-variant to a given style.
func MatchStyle(variantName string, style xfont.Style) MatchConfidence {
	variantName = strings.ToLower(variantName)
	switch style {
	case xfont.StyleNormal:
		switch variantName {
		case "regular", "400":
			return PerfectConfidence
		case "100", "200", "300", "500":
			return HighConfidence
		}
		return NoConfidence
	case xfont.StyleItalic:
		if strings.Contains(variantName, "italic") {
			return PerfectConfidence
		}
		if strings.Contains(variantName, "obliq") {
			return HighConfidence
		}
		return NoConfidence
	case xfont.StyleOblique:
		if strings.Contains(variantName, "obliq") {
			return PerfectConfidence
		}
		if strings.Contains(variantName, "italic") {
			return HighConfidence
		}
		return NoConfidence
	}
	return NoConfidence
}

// MatchWeight trys to match a font-variant to a given weight.
func MatchWeight(variantName string, weight xfont.Weight) MatchConfidence {
	/* from https://pkg.go.dev/golang.org/x/image/font
	WeightThin       Weight = -3 // CSS font-weight value 100.
	WeightExtraLight Weight = -2 // CSS font-weight value 200.
	WeightLight      Weight = -1 // CSS font-weight value 300.
	WeightNormal     Weight = +0 // CSS font-weight value 400.
	WeightMedium     Weight = +1 // CSS font-weight value 500.
	WeightSemiBold   Weight = +2 // CSS font-weight value 600.
	WeightBold       Weight = +3 // CSS font-weight value 700.
	WeightExtraBold  Weight = +4 // CSS font-weight value 800.
	WeightBlack      Weight = +5 // CSS font-weight value 900.
	*/
	if strconv.Itoa(int(weight)+4*100) == variantName {
		return PerfectConfidence
	}
	switch variantName {
	case "regular", "400", "italic", "oblique", "normal", "text":
		switch weight {
		case xfont.WeightNormal, xfont.WeightMedium:
			return PerfectConfidence
		case xfont.WeightThin, xfont.WeightExtraLight, xfont.WeightLight:
			return LowConfidence
		}
		return NoConfidence
	case "100", "200", "300":
		switch weight {
		case xfont.WeightThin, xfont.WeightExtraLight, xfont.WeightLight:
			return PerfectConfidence
		case xfont.WeightNormal, xfont.WeightMedium:
			return LowConfidence
		}
		return NoConfidence
	case "500":
		switch weight {
		case xfont.WeightMedium:
			return PerfectConfidence
		case xfont.WeightSemiBold:
			return HighConfidence
		case xfont.WeightNormal, xfont.WeightBold:
			return LowConfidence
		}
		return NoConfidence
	case "bold", "700":
		switch weight {
		case xfont.WeightBold:
			return PerfectConfidence
		case xfont.WeightSemiBold, xfont.WeightExtraBold:
			return HighConfidence
		}
		return NoConfidence
	case "extrabold", "600", "800", "900":
		switch weight {
		case xfont.WeightSemiBold:
			return LowConfidence
		case xfont.WeightBold:
			return HighConfidence
		}
		return NoConfidence
	}
	return NoConfidence
}
