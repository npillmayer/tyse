package resources

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"

	"github.com/npillmayer/schuko/gconf"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
)

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

func SetupGoogleFontsDirectory() error {
	loadGoogleFontsDir.Do(func() {
		apikey := gconf.GetString("google-api-key")
		if apikey == "" {
			apikey = os.Getenv("GOOGLE_API_KE")
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

// ---------------------------------------------------------------------------

// ListGoogleFonts produces a listing of available fonts from the Google webfont
// service, with font-family names matching a given pattern.
//
// If not aleady done, the list of fonts will be downloaded from Google.
func ListGoogleFonts(pattern string) {
	level := T().GetTraceLevel()
	T().SetTraceLevel(tracing.LevelInfo)
	if err := SetupGoogleFontsDirectory(); err != nil {
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
