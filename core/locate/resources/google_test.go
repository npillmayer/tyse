package resources

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
	xfont "golang.org/x/image/font"
)

// ATTENTION
// ---------
// Tests in this file require a Google Font Service API-key to be present.
// The API-key has to be set with the GOOGLE_API_KEY environment variable.

func TestAPIKeyPresent(t *testing.T) {
	if os.Getenv("GOOGLE_API_KEY") == "" {
		t.Fatalf(`

ATTENTION
---------
Tests in this file require a Google Font Service API-key to be present.
The API-key has to be set with the GOOGLE_API_KEY environment variable.

`)
	}
}

const exampleRespFragm string = `
{
    "kind": "webfonts#webfontList",
    "items": [
        {
            "kind": "webfonts#webfont",
            "family": "Anonymous Pro",
            "variants": [
                "regular",
                "italic",
                "700",
                "700italic"
            ],
            "subsets": [
                "greek",
                "greek-ext",
                "cyrillic-ext",
                "latin-ext",
                "latin",
                "cyrillic"
            ],
            "version": "v3",
            "lastModified": "2012-07-25",
            "files": {
                "regular": "http://themes.googleusercontent.com/static/fonts/anonymouspro/v3/Zhfjj_gat3waL4JSju74E-V_5zh5b-_HiooIRUBwn1A.ttf",
                "italic": "http://themes.googleusercontent.com/static/fonts/anonymouspro/v3/q0u6LFHwttnT_69euiDbWKwIsuKDCXG0NQm7BvAgx-c.ttf",
                "700": "http://themes.googleusercontent.com/static/fonts/anonymouspro/v3/WDf5lZYgdmmKhO8E1AQud--Cz_5MeePnXDAcLNWyBME.ttf",
                "700italic": "http://themes.googleusercontent.com/static/fonts/anonymouspro/v3/_fVr_XGln-cetWSUc-JpfA1LL9bfs7wyIp6F8OC9RxA.ttf"
            }
        },
        {
            "kind": "webfonts#webfont",
            "family": "Antic",
            "variants": [
                "regular"
            ],
            "subsets": [
                "latin"
            ],
            "version": "v4",
            "lastModified": "2012-07-25",
            "files": {
                "regular": "http://themes.googleusercontent.com/static/fonts/antic/v4/hEa8XCNM7tXGzD0Uk0AipA.ttf"
            }
        }
    ]
}
`

func TestGoogleRespDecode(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	dec := json.NewDecoder(strings.NewReader(exampleRespFragm))
	var list googleFontsList
	err := dec.Decode(&list)
	if err != nil {
		t.Fatal(err)
	}
	listGoogleFonts(list, ".*")
}

func TestGoogleAPI(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	err := setupGoogleFontsDirectory()
	if err != nil {
		core.UserError(err)
		t.Fatal(err)
	}
}

func TestMatchFontname(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	pattern := "Inconsolata"
	r, err := regexp.Compile(strings.ToLower(pattern))
	if err != nil {
		t.Fatal(err)
	}
	if !r.MatchString(strings.ToLower("Inconsolata")) {
		t.Errorf("expected to find match, didn't")
	}
}

func TestGoogleFindFont(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	fi, err := FindGoogleFont("Inconsolata", xfont.StyleNormal, xfont.WeightNormal)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fi {
		t.Logf("family = %s, variants = %+v", f.Family, f.Variants)
	}
	fi, err = FindGoogleFont("Inconsolata", xfont.StyleItalic, xfont.WeightNormal)
	if err == nil {
		t.Error("expected search for Inconsolata Italic to fail, did not")
	}
}

func TestGoogleCacheFont(t *testing.T) {
	teardown := testconfig.QuickConfig(t, map[string]string{
		"app-key": "tyse-test",
	})
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	fi, err := FindGoogleFont("Inconsolata", xfont.StyleNormal, xfont.WeightNormal)
	if err != nil {
		t.Fatal(err)
	}
	path, err := CacheGoogleFont(fi[0], "regular")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("path = %s", path)
	_, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
}
