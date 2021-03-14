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
	Family   string            `json:"family"`
	Version  string            `json:"version"`
	Variants []string          `json:"variants"`
	Subsets  []string          `json:"subsets"`
	Files    map[string]string `json:"files"`
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
		apikey := gconf.GetString("google-api-key")
		if apikey == "" {
			apikey = os.Getenv("GOOGLE_API_KEY")
		}
		if apikey == "" {
			err := errors.New("Google API key not set")
			T().Errorf(err.Error())
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
			T().Errorf("Google Fonts API request not OK: %s", err.Error())
			googleFontsLoadError = core.WrapError(err, core.ECONNECTION,
				"could not get fonts-diretory from Google font service")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			T().Errorf("Google Fonts API request not OK: %v", resp.Status)
			err := core.Error(resp.StatusCode, "response: %v", resp.Status)
			googleFontsLoadError = core.WrapError(err, core.ECONNECTION,
				"could not get fonts-diretory from Google font service")
			return
		}
		dec := json.NewDecoder(resp.Body)
		err = dec.Decode(&googleFontsDirectory)
		if err != nil {
			googleFontsLoadError = core.WrapError(err, core.EINVALID,
				"could not decode fonts-list from Google font service")
		}
	})
	return googleFontsLoadError
}

// FindGoogleFont scans the Google Font Service for fonts matching pattern, and
// having a given style and weight.
func FindGoogleFont(pattern string, style xfont.Style, weight xfont.Weight) ([]GoogleFontInfo, error) {
	var fi []GoogleFontInfo
	if err := setupGoogleFontsDirectory(); err != nil {
		return fi, err
	}
	r, err := regexp.Compile(pattern)
	if err != nil {
		return fi, core.WrapError(err, core.EINVALID,
			"cannot match Google font: invalid pattern: %v", err)
	}
	for _, finfo := range googleFontsDirectory.Items {
		if r.MatchString(finfo.Family) {
			match := false
			T().Debugf("Google font matches pattern: %s", finfo.Family)
			for _, v := range finfo.Variants {
				match = font.MatchStyle(v, style) && font.MatchWeight(v, weight)
				if match {
					fi = append(fi, finfo)
					break
				}
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
// The loaded is cached in the user's cache directory.
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
	T().Infof("caching font %s as %s", fi.Family, filepath)
	err = DownloadCachedFile(filepath, fileurl)
	if err != nil {
		return "", err
	}
	return filepath, nil
}

// ---------------------------------------------------------------------------

// ListGoogleFonts produces a listing of available fonts from the Google webfont
// service, with font-family names matching a given pattern.
//
// If not aleady done, the list of fonts will be downloaded from Google.
func ListGoogleFonts(pattern string) {
	level := T().GetTraceLevel()
	T().SetTraceLevel(tracing.LevelInfo)
	if err := setupGoogleFontsDirectory(); err != nil {
		T().Errorf(err.(core.AppError).UserMessage())
	} else {
		listGoogleFonts(googleFontsDirectory, pattern)
	}
	T().SetTraceLevel(level)
}

func listGoogleFonts(list googleFontsList, pattern string) {
	r, err := regexp.Compile(pattern)
	if err != nil {
		T().Errorf("cannot list Google fonts: invalid pattern: %v", err)
	}
	T().Infof("%d fonts in list", len(list.Items))
	T().Infof("======================================")
	for i, finfo := range list.Items {
		if r.MatchString(finfo.Family) {
			T().Infof("[%4d] %-20s: %s", i, finfo.Family, finfo.Version)
			T().Infof("       subsets: %v", finfo.Subsets)
			for k, v := range finfo.Files {
				T().Infof("       - %-18s: %s", k, v[len(v)-4:])
			}
		}
	}
}
