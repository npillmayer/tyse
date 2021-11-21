package resources

import (
	"context"
	"embed"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"io/ioutil"
	"path"
	"strings"

	"github.com/flopp/go-findfont"
	"github.com/npillmayer/schuko"
	"github.com/npillmayer/tyse/core"
	"github.com/npillmayer/tyse/core/font"
	xfont "golang.org/x/image/font"
)

type resourceType int

// resource types
const (
	unknownResourceType resourceType = iota
	fontResourceType
	imageResourceType
)

// notFound returns an application error for a missing resource.
func notFound(res string, rtype resourceType) error {
	e := fmt.Errorf("resouce missing: %v", res)
	var s string
	switch rtype {
	case imageResourceType:
		s = fmt.Sprintf("image not found: %s, loaded placeholder image instead", res)
	case fontResourceType:
		s = fmt.Sprintf("font not found: %s", res)
	default:
		s = fmt.Sprintf("resource not found: %s", res)
	}
	err := core.WrapError(e, core.EMISSING, s)
	return err
}

//go:embed packaged/*
var packaged embed.FS

// --- Images ---------------------------------------------------------------

type imgPlusErr struct {
	img image.Image
	err error
}

// ResolveImage currently will only search for images packaged with the
// application.
func ResolveImage(name string, resolution string) ImagePromise {
	ch := make(chan imgPlusErr)
	go func(ch chan<- imgPlusErr) {
		result := imgPlusErr{}
		images, _ := packaged.ReadDir("packaged/images")
		var imagename string
		for _, image := range images {
			//T().Debugf("image file %s", image.Name())
			if image.Name() == name {
				imagename = image.Name()
				break
			}
			if strings.HasPrefix(image.Name(), name+"-") {
				if strings.HasSuffix(image.Name(), resolution) {
					imagename = image.Name()
					break
				}
			}
		}
		if imagename == "" {
			imagename = "packaged/images/placeholder.png"
			result.err = notFound(name, imageResourceType)
		}
		file, err := packaged.Open("packaged/images/" + imagename)
		if err != nil {
			result.err = err
		} else {
			defer file.Close()
			result.img, err = png.Decode(file)
			if err != nil {
				result.err = err
			}
		}
		ch <- result
		close(ch)
	}(ch)
	return imageLoader{
		await: func(ctx context.Context) (image.Image, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case r := <-ch:
				return r.img, r.err
			}
		},
	}
}

// ImagePromise loads an image in the background. A call to `Image` will block
// until loading is completed.
type ImagePromise interface {
	Image() (image.Image, error)
}

type imageLoader struct {
	await func(ctx context.Context) (image.Image, error)
}

func (loader imageLoader) Image() (image.Image, error) {
	return loader.await(context.Background())
}

// --- Fonts -----------------------------------------------------------------

type fontPlusErr struct {
	font *font.TypeCase
	desc font.Descriptor
	err  error
}

// TypeCasePromise runs font location asynchronously in the background.
// A call to `TypeCase()` blocks until font loading is completed.
type TypeCasePromise interface {
	TypeCase() (*font.TypeCase, error)
	Descriptor() font.Descriptor // descriptor of typecase to load, before and after
}

type fontLoader struct {
	await func(ctx context.Context) (*font.TypeCase, error)
	desc  font.Descriptor
}

func (loader fontLoader) TypeCase() (*font.TypeCase, error) {
	return loader.await(context.Background())
}

func (loader fontLoader) Descriptor() font.Descriptor {
	return loader.desc
}

// ResolveTypeCase resolves a font typecase with a given size.
// It searches for fonts in the following order:
//
// ▪︎ Fonts packaged with the application binary
//
// ▪︎ System-fonts
//
// ▪︎ Google Fonts service (https://fonts.google.com/)
//
// ResolveTypeCase will try to match style and weight requirements closely, but
// will load a font variant anyway if it matches approximately. If, for example,
// a system contains a font with weight 300, which would be considered a "light"
// variant, but no variant with weight 400 (normal), it will load the 300-variant.
//
// When looking for sytem-fonts, ResolveTypeCase will use an existing fontconfig
// (https://www.freedesktop.org/wiki/Software/fontconfig/)
// installation, if present. fontconfig has to be configured in the global
// application setup by pointing to the absolute path of the `fc-list` binary.
// If fontconfig isn't installed or configured, then this step will silently be
// skipped and a file system scan of the sytem's fonts-folders will be done.
// (See also function `FindLocalFont`).
//
// A prerequisite to looking for Google fonts is a valid API-key (refer to
// https://developers.google.com/fonts/docs/developer_api). It has to be configured
// either in the application setup or as an environment variable GOOGLE_API_KEY.
// (See also function `FindGoogleFont`).
//
// If no suitable font can be found, an application-wide fallback font will be
// returned.
//
// Typecases are not returned synchronously, but rather as a promise
// of kind TypeCasePromise (async/await-pattern).
//
func ResolveTypeCase(conf schuko.Configuration, pattern string, style xfont.Style, weight xfont.Weight, size float64) TypeCasePromise {
	// TODO include a context parameter
	desc := font.Descriptor{
		Family: pattern,
	}
	ch := make(chan fontPlusErr)
	go func(ch chan<- fontPlusErr) {
		result := fontPlusErr{
			desc: desc,
		}
		name := font.NormalizeFontname(pattern, style, weight)
		if t, err := font.GlobalRegistry().TypeCase(name, size); err == nil {
			result.font = t
			result.desc.Family = t.ScalableFontParent().Fontname
			ch <- result
			close(ch)
			return
		}
		var f *font.ScalableFont
		fonts, _ := packaged.ReadDir("packaged/fonts")
		var fname string // path to embedded font, if any
		for _, f := range fonts {
			if font.Matches(f.Name(), pattern, style, weight) {
				tracer().Debugf("found embedded font file %s", f.Name())
				fname = f.Name()
				break
			}
		}
		if fname != "" { // font is packaged embedded font
			var file fs.File
			file, result.err = packaged.Open("packaged/fonts/" + fname)
			if result.err == nil {
				defer file.Close()
				bytez, _ := ioutil.ReadAll(file)
				if f, result.err = font.ParseOpenTypeFont(bytez); result.err == nil {
					result.desc.Family = fname
					name = fname
				}
			}
			if f == nil { // cannot process embedded font => seriously compromised installation
				result.err = core.WrapError(result.err, core.EINTERNAL,
					"internal application error - packaged font not readable: %s", fname)
				ch <- result
				close(ch)
				return
			}
		}
		if f == nil { // next try system fonts
			if desc, _ := FindLocalFont(conf, pattern, style, weight); desc.Family != "" {
				f, result.err = font.LoadOpenTypeFont(desc.Path)
			}
		}
		if f == nil { // next try Google font service
			var fiList []GoogleFontInfo
			if fiList, result.err = FindGoogleFont(conf, pattern, style, weight); result.err == nil {
				var l []font.Descriptor
				for _, finfo := range fiList { // morph Google font info font font.Descriptor list
					l = append(l, finfo.Descriptor)
				}
				desc, variant, confidence := font.ClosestMatch(l, pattern, style, weight)
				if confidence > font.LowConfidence {
					var fpath string
					var i int
					for j, d := range fiList { // find matching variant again
						if d.Descriptor.Family == desc.Family {
							i = j // this must succeed
						}
					}
					if fpath, result.err = CacheGoogleFont(fiList[i], variant); result.err == nil {
						f, result.err = font.LoadOpenTypeFont(fpath)
						name = path.Base(fpath)
						result.desc.Family = name
					}
				}
			}
		}
		if f != nil { // if found, enter into font registry
			f.Fontname = name
			font.GlobalRegistry().StoreFont(name, f)
			result.font, result.err = font.GlobalRegistry().TypeCase(name, size)
			result.desc.Family = name
			//font.GlobalRegistry().DebugList()
		} else { // use fallback font
			result.font, _ = font.GlobalRegistry().TypeCase("fallback", size)
			result.desc.Family = "fallback"
		}
		ch <- result
		close(ch)
	}(ch)
	loader := fontLoader{desc: desc}
	loader.await = func(ctx context.Context) (*font.TypeCase, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-ch:
			loader.desc = r.desc
			return r.font, r.err
		}
	}
	return loader
}

// FindLocalFont searches for a locally installed font variant.
//
// If present and configured, FindLocalFont will be using the fontconfig
// system (https://www.freedesktop.org/wiki/Software/fontconfig/).
// fontconfig has to be configured in the global application setup by
// pointing to the absolute path of the 'fc-list' binary.
//
// We will copy the output of fc-list to the user's config directory once.
// Subsequent calls will use the cached entries to search for
// a font, given a name pattern, a style and a weight.
// We call the binary instead of using the C library because of possible version
// issues and to reduce compile-time dependencies.
//
// If fontconfig is not configured, FindLocalFont will fall back to scanning
// the system's fonts-folders (OS dependent).
//
// (Please refer to function `ResolveTypeCase`, too)
//
func FindLocalFont(conf schuko.Configuration, pattern string, style xfont.Style, weight xfont.Weight) (
	desc font.Descriptor, variant string) {
	//
	desc, variant = findFontConfigFont(conf, pattern, style, weight)
	if desc.Family == "" {
		if loadedFontConfigListOK { // fontconfig is active, but didn't find a font
			return // don't do a file system scan
		}
	} // otherwise fontconfig is not active => scan file system
	fpath, err := findfont.Find(pattern) // lib does not accept style & weight
	if err == nil && fpath != "" {
		tracer().Debugf("%s is a system font: %s", pattern, fpath)
		desc = font.Descriptor{
			Family: font.NormalizeFontname(pattern, style, weight),
			Path:   fpath,
		}
	}
	return
}
