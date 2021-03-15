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
	err  error
}

// TypeCasePromise runs font location asynchronously in the background.
// A call to `TypeCase()` blocks until font loading is completed.
type TypeCasePromise interface {
	TypeCase() (*font.TypeCase, error)
}

type fontLoader struct {
	await func(ctx context.Context) (*font.TypeCase, error)
}

func (loader fontLoader) TypeCase() (*font.TypeCase, error) {
	return loader.await(context.Background())
}

// ResolveTypeCase resolves a font typecase with a given size.
// It searches for fonts in the following order:
//
// ▪︎ Fonts packaged with the application binary
//
// ▪︎ System fonts
//
// ▪︎ Google Fonts service (https://fonts.google.com/)
//
// ResolveTypeCase will try to match style and weight requirements closely, but
// will load a font variant anyway if it matches approximately. If, for example,
// a system contains a font with weight 300, which would be considered a "light"
// variant, but no variant with weight 400 (normal), it will load the 300-variant.
//
// If the user's application configuration contains an extended listing of system
// fonts (e.g., created by `tyseconfig`), ResolveTypeCase is able to match fonts
// reliably by name-pattern, style and weight. Otherwise it tries to derive style
// and weight information from the fonts' filenames.
//
func ResolveTypeCase(pattern string, style xfont.Style, weight xfont.Weight, size float64) TypeCasePromise {
	// TODO include a context parameter
	ch := make(chan fontPlusErr)
	go func(ch chan<- fontPlusErr) {
		result := fontPlusErr{}
		if t, err := font.GlobalRegistry().TypeCase(pattern, style, weight, size); err == nil {
			result.font = t
			ch <- result
			close(ch)
			return
		}
		fonts, _ := packaged.ReadDir("packaged/fonts")
		var f *font.ScalableFont
		var fname string
		for _, f := range fonts {
			trace().Debugf("font file %s", f.Name())
			if font.Matches(f.Name(), pattern, style, weight) {
				fname = f.Name()
				break
			}
		}
		name := font.NormalizeFontname(pattern, style, weight)
		if fname != "" { // found font as packaged embedded font
			trace().Debugf("found font as embedded font file %s", fname)
			var file fs.File
			file, result.err = packaged.Open("packaged/fonts/" + fname)
			if result.err == nil {
				defer file.Close()
				bytez, _ := ioutil.ReadAll(file)
				if f, result.err = font.ParseOpenTypeFont(bytez); result.err == nil {
					name = fname
				}
			}
		}
		if f == nil { // next try system fonts
			fpath, err := findfont.Find(pattern) // try to find as system font
			if err == nil && fpath != "" {
				trace().Debugf("%s is a system font: %s", pattern, fpath)
				f, result.err = font.LoadOpenTypeFont(fpath)
			}
		}
		if f == nil { // next try Google font service
			var fiList []GoogleFontInfo
			if fiList, result.err = FindGoogleFont(pattern, style, weight); result.err == nil {
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
					}
				}
			}
		}
		//font.GlobalRegistry().DebugList()
		if f != nil {
			f.Fontname = name
			font.GlobalRegistry().StoreFont(f)
			result.font, result.err = font.GlobalRegistry().TypeCase(name, style, weight, size)
			//font.GlobalRegistry().DebugList()
		}
		ch <- result
		close(ch)
	}(ch)
	return fontLoader{
		await: func(ctx context.Context) (*font.TypeCase, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case r := <-ch:
				return r.font, r.err
			}
		},
	}
}
