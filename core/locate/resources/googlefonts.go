package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/npillmayer/schuko/gconf"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
	"github.com/npillmayer/tyse/core/font"
	xfont "golang.org/x/image/font"
)

// GoogleFontInfo describes a font entry in the Google Font Service.
type GoogleFontInfo struct {
	font.Descriptor
	Version string            `json:"version"`
	Subsets []string          `json:"subsets"`
	Files   map[string]string `json:"files"`
}

type googleFontsList struct {
	Items []GoogleFontInfo `json:"items"`
}

var loadGoogleFontsDir sync.Once
var googleFontsDirectory googleFontsList
var googleFontsLoadError error
var googleFontsAPI string = `https://www.googleapis.com/webfonts/v1/webfonts?`

func setupGoogleFontsDirectory() error {
	loadGoogleFontsDir.Do(func() {
		trace().Infof("setting up Google Fonts service directory")
		apikey := gconf.GetString("google-api-key")
		if apikey == "" {
			apikey = os.Getenv("GOOGLE_API_KEY")
		}
		if apikey == "" {
			err := errors.New("Google API key not set")
			trace().Errorf(err.Error())
			googleFontsLoadError = core.WrapError(err, core.EMISSING,
				`Google Fonts API-key must be set in global configuration or as GOOGLE_API_KEY in environment;
      please refer to https://developers.google.com/fonts/docs/developer_api`)
			return
		}
		values := url.Values{
			"sort": []string{"alpha"},
			"key":  []string{apikey},
		}
		resp, err := http.Get(googleFontsAPI + values.Encode())
		if err != nil {
			trace().Errorf("Google Fonts API request not OK: %s", err.Error())
			googleFontsLoadError = core.WrapError(err, core.ECONNECTION,
				"could not get fonts-diretory from Google font service")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			trace().Errorf("Google Fonts API request not OK: %v", resp.Status)
			err := core.Error(resp.StatusCode, "response: %v", resp.Status)
			googleFontsLoadError = core.WrapError(err, core.ECONNECTION,
				"could not get fonts-diretory from Google font service")
			return
		}
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&googleFontsDirectory)
		if err != nil {
			trace().Errorf("Google Fonts API response not decoded: %v", err)
			googleFontsLoadError = core.WrapError(err, core.EINVALID,
				"could not decode fonts-list from Google font service")
		}
		trace().Infof("transfered list of %d font from Google Fonts service",
			len(googleFontsDirectory.Items))
	})
	return googleFontsLoadError
}

// FindGoogleFont scans the Google Font Service for fonts matching `pattern` and
// having a given style and weight.
//
// Will include all fonts with a match-confidence greater than `font.LowConfidence`.
//
// A prerequisite to looking for Google fonts is a valid API-key (refer to
// https://developers.google.com/fonts/docs/developer_api). It has to be configured
// either in the application setup or as an environment variable GOOGLE_API_KEY.
//
// (Please refer to function `ResolveTypeCase`, too)
//
func FindGoogleFont(pattern string, style xfont.Style, weight xfont.Weight) ([]GoogleFontInfo, error) {
	var fi []GoogleFontInfo
	if err := setupGoogleFontsDirectory(); err != nil {
		return fi, err
	}
	r, err := regexp.Compile(strings.ToLower(pattern))
	if err != nil {
		return fi, core.WrapError(err, core.EINVALID,
			"cannot match Google font: invalid font name pattern: %v", err)
	}
	//trace().Debugf("trying to match (%s)", strings.ToLower(pattern))
	for _, finfo := range googleFontsDirectory.Items {
		//trace().Debugf("testing (%s)", strings.ToLower(finfo.Family))
		if r.MatchString(strings.ToLower(finfo.Family)) {
			trace().Debugf("Google font name matches pattern: %s", finfo.Family)
			_, _, conf := font.ClosestMatch([]font.Descriptor{finfo.Descriptor}, pattern,
				style, weight)
			if conf > font.LowConfidence {
				fi = append(fi, finfo)
				break
			}
		}
	}
	if len(fi) == 0 {
		return fi, errors.New("no Google font matches pattern")
	}
	//T().Debugf("found %v", fi[0])
	return fi, nil
}

// ---------------------------------------------------------------------------

// CacheGoogleFont loads a font described by fi with a given variant.
// The loaded font is cached in the user's cache directory.
func CacheGoogleFont(fi GoogleFontInfo, variant string) (filepath string, err error) {
	var fileurl string
	for _, v := range fi.Variants {
		if v == variant {
			fileurl = fi.Files[v]
		}
	}
	if fileurl == "" {
		return "", fmt.Errorf("no variant equals %s, cannot cache %s", variant, fi.Family)
	}
	letter := strings.ToUpper(fi.Family[:1])
	cachedir, err := CacheDirPath("fonts", letter)
	if err != nil {
		return "", err
	}
	ext := path.Ext(fileurl)
	name := fi.Family + "-" + variant + ext
	filepath = path.Join(cachedir, name)
	trace().Infof("caching font %s as %s", fi.Family, filepath)
	if _, err := os.Stat(filepath); err == nil {
		trace().Infof("font already cached: %s", filepath)
	} else {
		err = DownloadCachedFile(filepath, fileurl)
		if err != nil {
			return "", err
		}
	}
	return filepath, nil
}

// ---------------------------------------------------------------------------

// ListGoogleFonts produces a listing of available fonts from the Google webfont
// service, with font-family names matching a given pattern.
// Output goes into the trace file with log-level info.
//
// If not aleady done, the list of available fonts will be downloaded from Google.
func ListGoogleFonts(pattern string) {
	level := trace().GetTraceLevel()
	trace().SetTraceLevel(tracing.LevelInfo)
	if err := setupGoogleFontsDirectory(); err != nil {
		trace().Errorf(err.(core.AppError).UserMessage())
	} else {
		listGoogleFonts(googleFontsDirectory, pattern)
	}
	trace().SetTraceLevel(level)
}

func listGoogleFonts(list googleFontsList, pattern string) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		trace().Errorf("cannot list Google fonts: invalid pattern: %v", err)
	}
	trace().Infof("%d fonts in list", len(list.Items))
	trace().Infof("======================================")
	for i, finfo := range list.Items {
		if r.MatchString(finfo.Family) {
			trace().Infof("[%4d] %-20s: %s", i, finfo.Family, finfo.Version)
			trace().Infof("       subsets: %v", finfo.Subsets)
			for k, v := range finfo.Files {
				trace().Infof("       - %-18s: %s", k, v[len(v)-4:])
			}
		}
	}
}
