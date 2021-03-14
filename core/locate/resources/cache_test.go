package resources

import (
	"path"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
)

func TestCacheDownload(t *testing.T) {
	teardown := testconfig.QuickConfig(t, map[string]string{
		"app-key": "tyse-test",
	})
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	cachedir, err := CacheDirPath("fonts")
	if err != nil {
		t.Fatal(err)
	}
	err = DownloadCachedFile(path.Join(cachedir, "test.svg"),
		"https://npillmayer.github.io/UAX/img/UAX-Logo-shadow.svg")
	if err != nil {
		t.Fatal(err)
	}
}
