package resources

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/npillmayer/schuko"
	"github.com/npillmayer/tyse/core"
	"github.com/npillmayer/tyse/core/font"
	xfont "golang.org/x/image/font"
)

func findFontConfigBinary(conf schuko.Configuration) (path string, err error) {
	path = conf.GetString("fontconfig")
	if path == "" {
		tracer().Infof("fontconfig not configured: key 'fontconfig' should point location of 'fc-list' binary")
		err = errors.New("fontconfig not configured")
	}
	return
}

func cacheFontConfigList(conf schuko.Configuration, update bool) (string, bool) {
	appkey := conf.GetString("app-key")
	tracer().Debugf("config[app-key] = %s", appkey)
	uconfdir, err := os.UserConfigDir()
	if appkey == "" || err != nil {
		tracer().Errorf("user config directory not set")
		return "", false
	}
	fcListFilename := path.Join(uconfdir, appkey, "fontlist.txt")
	if _, err := os.Stat(fcListFilename); err == nil {
		// fontlist already exists
		if !update {
			return fcListFilename, true
		}
	} else { // create config sub-dir for this application
		dir := path.Join(uconfdir, appkey)
		if _, err = os.Stat(dir); os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				err = core.WrapError(err, core.EINVALID,
					"user configuration path cannot be created: %s", dir)
				core.UserError(err)
				return "", false
			}
		}
	}
	fcpath, err := findFontConfigBinary(conf)
	if err != nil {
		return "", false
	}
	if !path.IsAbs(fcpath) {
		err = core.Error(core.EINVALID, "fontconfig binary fc-list must point to absolute path: %s", fcpath)
		core.UserError(err)
		return "", false
	}
	if fi, err := os.Stat(fcpath); err != nil || (fi.Mode().Perm()&0100) == 0 {
		err = core.WrapError(err, core.EINVALID,
			"fontconfig configuration points to an invalid binary: %s", fcpath)
		core.UserError(err)
		return "", false
	}
	fontlistFile, err := os.Create(fcListFilename)
	if err == nil {
		fccmd := exec.Command(fcpath)
		fccmd.Stdout = fontlistFile
		err = fccmd.Run()
	}
	if err != nil {
		err = core.WrapError(err, core.EINVALID,
			"fontconfig output file cannot be created: %s", fcListFilename)
		core.UserError(err)
		return "", false
	}
	return fcListFilename, true
}

func loadFontConfigList(conf schuko.Configuration) ([]font.Descriptor, bool) {
	fclist, ok := cacheFontConfigList(conf, false)
	if !ok {
		return []font.Descriptor{}, false
	}
	fc, err := os.Open(fclist)
	if err != nil {
		err = core.WrapError(err, core.EINVALID,
			"fontconfig font list cannot be opened: %s", fclist)
		core.UserError(err)
		return []font.Descriptor{}, false
	}
	defer fc.Close()
	scanner := bufio.NewScanner(fc)
	ttc := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 3 {
			continue
		}
		fontpath := strings.TrimSpace(fields[0])
		fontname := strings.TrimSpace(fields[1])
		fontname = strings.TrimPrefix(fontname, ".")
		fontvari := strings.ToLower(fields[2])
		if strings.HasSuffix(fontpath, ".ttc") {
			ttc++
			continue
		}
		desc := font.Descriptor{
			Family: fontname,
			Path:   fontpath,
		}
		if strings.Contains(fontvari, "regular") {
			desc.Variants = []string{"regular"}
		} else if strings.Contains(fontvari, "text") {
			desc.Variants = []string{"regular"}
		} else if strings.Contains(fontvari, "light") {
			desc.Variants = []string{"light"}
		} else if strings.Contains(fontvari, "italic") {
			desc.Variants = []string{"italic"}
		} else if strings.Contains(fontvari, "bold") {
			desc.Variants = []string{"bold"}
		} else if strings.Contains(fontvari, "black") {
			desc.Variants = []string{"bold"}
		}
		fontConfigDescriptors = append(fontConfigDescriptors, desc)
	}
	if err = scanner.Err(); err != nil {
		err = core.WrapError(err, core.EINVALID,
			"encountered a problem during reading of fontconfig font list: %s", fclist)
		core.UserError(err)
		return fontConfigDescriptors, false
	}
	if ttc > 0 {
		tracer().Infof("skipping %d platform fonts: TTC not yet supported", ttc)
	}
	return fontConfigDescriptors, true
}

var loadFontConfigListTask sync.Once
var loadedFontConfigListOK bool
var fontConfigDescriptors []font.Descriptor

// findFontConfigFont searches for a locally installed font variant using the fontconfig
// system (https://www.freedesktop.org/wiki/Software/fontconfig/).
// fontconfig has to be configured in the global application configuration by
// setting the absolute path of the 'fc-list' binary.
//
// FindFontConfigFont will copy the output of fc-list to the user's config
// directory once. Subsequent calls will use the cached entries to search for
// a font, given a name pattern, a style and a weight.
//
// We call the binary instead of using the C library because of possible version
// issues. If fontconfig is not configured, FindFontConfigFont will silently return an
// empty font descriptor and an empty variant name.
//
func findFontConfigFont(conf schuko.Configuration, pattern string, style xfont.Style, weight xfont.Weight) (
	desc font.Descriptor, variant string) {
	//
	loadFontConfigListTask.Do(func() {
		_, loadedFontConfigListOK = loadFontConfigList(conf)
		tracer().Infof("loaded fontconfig list")
	})
	if !loadedFontConfigListOK {
		return
	}
	var confidence font.MatchConfidence
	desc, variant, confidence = font.ClosestMatch(fontConfigDescriptors, pattern, style, weight)
	tracer().Debugf("closest fontconfig match confidence for %s|%s= %d", desc.Family, variant, confidence)
	if confidence > font.LowConfidence {
		return
	}
	return font.Descriptor{}, ""
}
