/*
Package font is for typeface and font handling.

There is a certain confusion in the nomenclature of typesetting. We will
stick to the following definitions:

* A "typeface" is a family of fonts. An example is "Helvetica".
This corresponds to a TrueType "collection" (*.ttc).

* A "scalable font" is a font, i.e. a variant of a typeface with a
certain weight, slant, etc.  An example is "Helvetica regular".

* A "typecase" is a scaled font, i.e. a font in a certain size for
a certain script and language. The name is reminiscend on the wooden
boxes of typesetters in the aera of metal type.
An example is "Helvetica regular 11pt, Latin, en_US".

Please note that Go (Golang) does use the terms "font" and "face"
differently–actually more or less in an opposite manner.

TODO: font collections (*.ttc), e.g., /System/Library/Fonts/Helvetica.ttc

Eigenen Text-Processor schreiben, nur für Latin Script, in pur Go?
Alternative zu Harfbuzz; also Latin-Harfbuzz für Arme in Go?
Siehe
https://docs.microsoft.com/en-us/typography/opentype/spec/ttochap1#text-processing-with-opentype-layout-fonts

Utility to view a character map of a font: http://torinak.com/font/lsfont.html

Website for fonts:
https://www.fontsquirrel.com/fonts/list/popular

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

3. Neither the name of this software nor the names of its contributors
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

// T traces to a global core-tracer.
func T() tracing.Trace {
	return gtrace.CoreTracer
}

type ScalableFont struct {
	Fontname string
	Filepath string     // file path
	Binary   []byte     // raw data
	SFNT     *sfnt.Font // the font's container // TODO: not threadsafe???
}

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

func LoadOpenTypeFont(fontfile string) (*ScalableFont, error) {
	bytez, err := ioutil.ReadFile(fontfile)
	if err != nil {
		return nil, err
	}
	return ParseOpenTypeFont(bytez)
}

func ParseOpenTypeFont(fbytes []byte) (f *ScalableFont, err error) {
	f = &ScalableFont{Binary: fbytes}
	f.SFNT, err = sfnt.Parse(f.Binary)
	if err != nil {
		return nil, err
	}
	f.Fontname, _ = f.SFNT.Name(nil, sfnt.NameIDFull)
	return
}

// TODO: check if language fits to script
// TODO: check if font supports script
func (sf *ScalableFont) PrepareCase(fontsize float64) (*TypeCase, error) {
	typecase := &TypeCase{}
	typecase.scalableFontParent = sf
	if fontsize < 5.0 || fontsize > 500.0 {
		fmt.Printf("*** font size must be 5pt < size < 500pt, is %g (set to 10pt)\n", fontsize)
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

func (tc *TypeCase) ScalableFontParent() *ScalableFont {
	return tc.scalableFontParent
}

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

type Registry struct {
	sync.Mutex
	fonts     map[string]*ScalableFont
	typecases map[string]*TypeCase
}

var globalFontRegistry *Registry

var globalRegistryCreation sync.Once

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

func (fr *Registry) StoreFont(f *ScalableFont) {
	if f == nil {
		T().Errorf("registry cannot store null font")
		return
	}
	fr.Lock()
	defer fr.Unlock()
	fname := NormalizeFontname(f.Fontname)
	T().Debugf("registry stores font %s as %s", f.Fontname, fname)
	fr.fonts[fname] = f
}

func (fr *Registry) TypeCase(name string, size float64) (*TypeCase, error) {
	T().Debugf("registry searches for font %s at %.2f", name, size)
	fname := NormalizeFontname(name)
	tname := NormalizeTypeCaseName(name, size)
	fr.Lock()
	defer fr.Unlock()
	if t, ok := fr.typecases[tname]; ok {
		T().Debugf("registry found font %s", tname)
		return t, nil
	}
	if f, ok := fr.fonts[fname]; ok {
		t, err := f.PrepareCase(size)
		T().Infof("font registry has font %s, caches at %.2f", fname, size)
		t.scalableFontParent = f
		fr.typecases[tname] = t
		return t, err
	}
	T().Infof("registry does not contain font %s", name)
	err := errors.New("font " + name + " not found in registry")
	fname = NormalizeTypeCaseName("fallback", size)
	tname = NormalizeTypeCaseName("fallback", size)
	if t, ok := fr.typecases[fname]; ok {
		return t, err
	}
	f := FallbackFont()
	t, _ := f.PrepareCase(size)
	T().Infof("font registry caches fallback font %s at %.2f", fname, size)
	fr.fonts[fname] = f
	fr.typecases[tname] = t
	return t, err
}

func (fr *Registry) DebugList() {
	T().Debugf("--- registered fonts ---")
	for k, v := range fr.fonts {
		T().Debugf("font [%s] = %v", k, v.Fontname)
	}
	for k, v := range fr.typecases {
		T().Debugf("typecase [%s] = %v", k, v.scalableFontParent.Fontname)
	}
	T().Debugf("------------------------")
}

func NormalizeFontname(fname string) string {
	fname = strings.TrimSpace(fname)
	fname = strings.ReplaceAll(fname, " ", "_")
	if dot := strings.LastIndex(fname, "."); dot > 0 {
		fname = fname[:dot]
	}
	fname = strings.ToLower(fname)
	return fname
}

func NormalizeTypeCaseName(fname string, size float64) string {
	fname = NormalizeFontname(fname)
	fname = fmt.Sprintf("%s-%.2f", fname, size)
	return fname
}

// ---------------------------------------------------------------------------

func MatchStyle(variantName string, style xfont.Style) bool {
	switch style {
	case xfont.StyleNormal:
		switch variantName {
		case "regular", "100", "200", "300", "400", "500":
			return true
		}
		return false
	case xfont.StyleItalic, xfont.StyleOblique:
		switch variantName {
		case "italic", "100italic", "200italic", "300italic", "400italic", "500italic":
			return true
		}
		return false
	}
	return false
}

func MatchWeight(variantName string, weight xfont.Weight) bool {
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
		return true
	}
	switch variantName {
	case "regular", "100", "200", "300", "400", "500":
		switch weight {
		case xfont.WeightThin, xfont.WeightExtraLight, xfont.WeightLight, xfont.WeightNormal, xfont.WeightMedium:
			return true
		}
		return false
	case "bold", "extrabold", "600", "700", "800", "900":
		switch weight {
		case xfont.WeightSemiBold, xfont.WeightBold, xfont.WeightExtraBold, xfont.WeightBlack:
			return true
		}
		return false
	}
	return false
}
