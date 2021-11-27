package fontregistry

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/font"
	xfont "golang.org/x/image/font"
)

// Registry is a type for holding information about loaded fonts for a
// typesetter.
type Registry struct {
	sync.Mutex
	fonts     map[string]*font.ScalableFont
	typecases map[string]*font.TypeCase
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
		fonts:     make(map[string]*font.ScalableFont),
		typecases: make(map[string]*font.TypeCase),
	}
	return fr
}

// StoreFont pushes a font into the registry if it isn't contained yet.
//
// The font will be stored using the normalized font name as a key. If this
// key is already associated with a font, that font will not be overridden.
func (fr *Registry) StoreFont(normalizedName string, f *font.ScalableFont) {
	if f == nil {
		tracer().Errorf("registry cannot store null font")
		return
	}
	fr.Lock()
	defer fr.Unlock()
	//style, weight := GuessStyleAndWeight(f.Fontname)
	//fname := NormalizeFontname(f.Fontname, style, weight)
	if _, ok := fr.fonts[normalizedName]; !ok {
		tracer().Debugf("registry stores font %s as %s", f.Fontname, normalizedName)
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
func (fr *Registry) TypeCase(normalizedName string, size float32) (*font.TypeCase, error) {
	//
	tracer().Debugf("registry searches for font %s at %.2f", normalizedName, size)
	//fname := NormalizeFontname(name, style, weight)
	tname := appendSize(normalizedName, size)
	fr.Lock()
	defer fr.Unlock()
	if t, ok := fr.typecases[tname]; ok {
		tracer().Infof("registry found font %s", tname)
		return t, nil
	}
	if f, ok := fr.fonts[normalizedName]; ok {
		t, err := f.PrepareCase(size)
		tracer().Infof("font registry has font %s, caches at %.2f", normalizedName, size)
		fr.typecases[tname] = t
		return t, err
	}
	tracer().Infof("registry does not contain font %s", normalizedName)
	err := errors.New("font " + normalizedName + " not found in registry")
	//
	// store typecase from fallback font, if not present yet, and return it
	fname := "fallback"
	tname = appendSize("fallback", size)
	if t, ok := fr.typecases[fname]; ok {
		return t, err
	}
	f := font.FallbackFont()
	t, _ := f.PrepareCase(size)
	tracer().Infof("font registry caches fallback font %s at %.2f", fname, size)
	fr.fonts[fname] = f
	fr.typecases[tname] = t
	return t, err
}

// LogFontList is a helper function to dump the list of known fonts and typecases
// in a registry to the trace-file (log-level Info).
func (fr *Registry) LogFontList() {
	level := tracer().GetTraceLevel()
	tracer().SetTraceLevel(tracing.LevelInfo)
	tracer().Infof("--- registered fonts ---")
	for k, v := range fr.fonts {
		tracer().Infof("font [%s] = %v", k, v.Fontname)
	}
	for k, v := range fr.typecases {
		tracer().Infof("typecase [%s] = %v", k, v.ScalableFontParent().Fontname)
	}
	tracer().Infof("------------------------")
	tracer().SetTraceLevel(level)
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

func appendSize(fname string, size float32) string {
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
	tracer().Debugf("basename of font = %s", basename)
	if !strings.Contains(basename, strings.ToLower(pattern)) {
		return false
	}
	s, w := GuessStyleAndWeight(basename)
	if s == style && w == weight {
		return true
	}
	return false
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
func ClosestMatch(fdescs []font.Descriptor, pattern string, style xfont.Style,
	weight xfont.Weight) (match font.Descriptor, variant string, confidence MatchConfidence) {
	//
	r, err := regexp.Compile(strings.ToLower(pattern))
	if err != nil {
		tracer().Errorf("invalid font name pattern")
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
