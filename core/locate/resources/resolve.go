package resources

import (
	"context"
	"embed"
	"fmt"
	"image"
	"image/png"
	"io/fs"
	"io/ioutil"
	"strings"

	"github.com/flopp/go-findfont"
	"github.com/npillmayer/tyse/core"
	"github.com/npillmayer/tyse/core/font"
)

type resourceType int

// resource types
const (
	unknownResourceType resourceType = iota
	fontResourceType
	imageResourceType
)

// NotFound returns an application error for a missing resource.
func NotFound(res string, rtype resourceType) error {
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
			result.err = NotFound(name, imageResourceType)
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

type TypeCasePromise interface {
	TypeCase() (*font.TypeCase, error)
}

type fontLoader struct {
	await func(ctx context.Context) (*font.TypeCase, error)
}

func (loader fontLoader) TypeCase() (*font.TypeCase, error) {
	return loader.await(context.Background())
}

func ResolveTypeCase(name string, size float64) TypeCasePromise {
	ch := make(chan fontPlusErr)
	go func(ch chan<- fontPlusErr) {
		result := fontPlusErr{}
		if t, err := font.GlobalRegistry().TypeCase(name, size); err == nil {
			result.font = t
			ch <- result
			close(ch)
			return
		}
		fonts, _ := packaged.ReadDir("packaged/fonts")
		var f *font.ScalableFont
		var fname string
		for _, f := range fonts {
			T().Debugf("font file %s", f.Name())
			if font.NormalizeFontname(f.Name()) == font.NormalizeFontname(name) {
				fname = f.Name()
				break
			}
		}
		if fname != "" { // found font as packaged embedded font
			T().Debugf("found font as embedded font file %s", fname)
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
		if f == nil {
			fpath, err := findfont.Find(name) // try to find as system font
			if err == nil && fpath != "" {
				T().Debugf("%s is a system font", name)
				f, result.err = font.LoadOpenTypeFont(fpath)
			}
		}
		if f == nil {
			// TODO load font from network
		}
		//font.GlobalRegistry().DebugList()
		if f != nil {
			f.Fontname = name
			font.GlobalRegistry().StoreFont(f)
			result.font, result.err = font.GlobalRegistry().TypeCase(name, size)
			//font.GlobalRegistry().DebugList()
			//T().Debugf("name = %v", result.font.ScalableFontParent().Fontname)
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
