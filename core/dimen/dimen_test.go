package dimen

import (
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestParseDimen(t *testing.T) {
	teardown := gotestingadapter.QuickConfig(t, "tyse.core")
	defer teardown()
	//
	d, _, err := ParseDimen("12px")
	if err != nil {
		t.Errorf("(1) %s", err.Error())
	} else if d != 12*BP {
		t.Errorf("(1) expected d to be 12bp (%d), is %d", 12*BP, d)
	}
	//
	d, _, err = ParseDimen("0")
	if err != nil {
		t.Errorf("(2) %s", err.Error())
	} else if d != 0 {
		t.Errorf("(2) expected d to be 0, is %d", d)
	}
	//
	d, ispcnt, err := ParseDimen("20%")
	if err != nil {
		t.Errorf("(3) %s", err.Error())
	} else if ispcnt != true {
		t.Errorf("(3) expected percentage-marker to be true, is %v", ispcnt)
	}
}
