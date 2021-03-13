package font

import (
	"fmt"
	"testing"
)

func TestOpenOpenTypeCaseCreation(t *testing.T) {
	//fontpath := locate.FileResource("GentiumPlus-R.ttf", "font")
	fontpath := "../locate/resources/packaged/fonts/GentiumPlus-R.ttf"
	f, err := LoadOpenTypeFont(fontpath)
	if err != nil {
		t.Fatal(err)
	}
	tc, err := f.PrepareCase(12.0)
	if err != nil {
		t.Logf("cannot create OT face for [%s]\n", f.Fontname)
		t.Fatal(err)
	}
	metrics := tc.font.Metrics()
	fmt.Printf("interline spacing for [%s]@%.1fpt is %s\n", f.Fontname, tc.size, metrics.Height)
}
