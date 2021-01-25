package font

import (
	"fmt"
	"testing"

	"github.com/npillmayer/tyse/core/locate"
)

func TestOpenOpenTypeLoading(t *testing.T) {
	fontpath := locate.FileResource("GentiumPlus-R.ttf", "font")
	f, err := LoadOpenTypeFont(fontpath)
	if err == nil {
		fmt.Printf("loaded font [%s] from \"%s\"\n", f.Fontname, fontpath)
	} else {
		t.Fatalf(err.Error())
	}
}

func TestOpenOpenTypeCaseCreation(t *testing.T) {
	fontpath := locate.FileResource("GentiumPlus-R.ttf", "font")
	f, err := LoadOpenTypeFont(fontpath)
	if err != nil {
		t.Fail()
	}
	tc, err2 := f.PrepareCase(12.0)
	if err2 != nil {
		fmt.Printf("cannot create OT face for [%s]\n", f.Fontname)
		t.Fatalf(err2.Error())
	}
	metrics := tc.font.Metrics()
	fmt.Printf("interline spacing for [%s]@%.1fpt is %s\n", f.Fontname, tc.size, metrics.Height)
}
