package resources

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/npillmayer/schuko/gconf"
	"github.com/npillmayer/tyse/core"
)

// DownloadFile will download a url to a local file (usually located in the
// user's cache directory).
func DownloadCachedFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// CacheDirPath checks and possibly creates a folder in the user's cache
// directory. The base cache directory is taken from `os.UserCacheDir()`, plus
// an application specific key, taken as `app-key` from the global configuration.
// Clients may specify a sequence of folder names, which will be appended to
// the base cache path. Non-existing sub-folders will be created as necessary
// (with permissions 755).
func CacheDirPath(subfolders ...string) (string, error) {
	T().Debugf("config[%s] = %s", "app-key", gconf.GetString("app-key"))
	if gconf.GetString("app-key") == "" {
		return "", core.WrapError(errors.New("application key is not set"), core.EMISSING,
			"application key is not configured; need to set it for cache access")
	}
	cachedir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	subs := path.Join(subfolders...)
	cachedir = path.Join(cachedir, gconf.GetString("app-key"), subs)
	T().Infof("caching in %s", cachedir)
	_, err = os.Stat(cachedir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(cachedir, 0755)
		if err != nil {
			return "", err
		}
	}
	return cachedir, nil
}
