package option_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/testconfig"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core/option"
)

func TestOptionMaybe(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var y1, y2, y3 interface{}
	x := option.SomeInt64(42)
	t.Logf("x = %v, x.T = %T, x.unwrap = %v", x, x, x.Unwrap())
	y1, _ = x.Match(option.Maybe{
		option.None: 7,
		option.Some: x.Unwrap() + 1,
	})
	//
	x = option.Int64()
	y2, _ = x.Match(option.Maybe{
		option.None: "No Value",
		option.Some: stringify,
	})
	//
	x = option.SomeInt64(42)
	y3, _ = x.Match(option.Maybe{
		option.None:  "No Value",
		option.Some:  nonsense,
		option.Error: stringify,
	})
	//
	t.Logf("y1 = %d, y2 = %s, y3 = %v", y1, y2, y3)
	if y1.(int64) != 43 {
		t.Errorf("expected SomeInt(42) to match to 43, is %d", y1)
	}
	if y2.(string) != "No Value" {
		t.Errorf("expected null-int64 to match to No Value, is %v", y2)
	}
	if y3 != "Value = 42" {
		t.Errorf("expected SomeInt(42) to match to Value = 42, is %v", y3)
	}
}

func TestOptionOf(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var y1 interface{}
	x := option.SomeInt64(1)
	t.Logf("x = %v, x.T = %T, x.unwrap = %v", x, x, x.Unwrap())
	y1, _ = x.Match(option.Of{
		option.None: 7,
		1:           99,
		option.Some: x.Unwrap(),
	})
	//
	t.Logf("y1 = %d", y1)
	if y1.(int) != 99 {
		t.Errorf("expected SomeInt(42) to match to 99, is %d", y1)
	}
}

func TestOptionRef(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	var y1 interface{}
	x := option.Something("hey")
	t.Logf("x = %v, x.T = %T, x.unwrap = %v", x, x, x.Unwrap())
	y1, _ = x.Match(option.Of{
		option.None: 0,
		"hey":       99,
		option.Some: 1,
	})
	//
	t.Logf("y1 = %d", y1)
	if y1.(int) != 99 {
		t.Errorf("expected Something(hey) to match to 99, is %d", y1)
	}
}

func TestOptionFail(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	x := option.SomeInt64(1)
	t.Logf("x = %v, x.T = %T, x.unwrap = %v", x, x, x.Unwrap())
	_, err := x.Match(option.Of{
		option.None:  7,
		1:            option.Fail(errors.New("Fail")),
		option.Some:  x.Unwrap(),
		option.Error: option.Fail(errors.New("Caught Fail")),
	})
	//
	t.Logf("err = %v", err)
	if err == nil {
		t.Errorf("expected SomeInt(1) to match to an error, hasn't")
	}
	if err.Error() != "Caught Fail" {
		t.Errorf("expected SomeInt(1) error to be caught, isn't")
	}
}

func TestOptionWrap(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	x := option.SomeInt64(1)
	t.Logf("x = %v, x.T = %T, x.unwrap = %v", x, x, x.Unwrap())
	s, err := x.Match(option.Of{
		option.None: "None",
		option.Some: option.WrapResult(stringify(x.Unwrap())),
	})
	//
	t.Logf("s = %+v, err = %v", s, err)
	if err != nil {
		t.Errorf("expected SomeInt(1) to match without error, hasn't")
	}
	if s == nil {
		t.Errorf("expected int64(1) to match to non-nil result, didn't")
	}
	if str, ok := s.(string); ok {
		if str != "Value = 1" {
			t.Errorf("expected int64(1) to match to 'Value = 1', didn't")
		}
	} else {
		t.Errorf("expected int64(1) to match to string, didn't")
	}
}

func TestOptionWrapError(t *testing.T) {
	teardown := testconfig.QuickConfig(t)
	defer teardown()
	gtrace.CoreTracer.SetTraceLevel(tracing.LevelDebug)
	//
	x := option.SomeInt64(1)
	t.Logf("x = %v, x.T = %T, x.unwrap = %v", x, x, x.Unwrap())
	s, err := x.Match(option.Of{
		option.None:  "None",
		option.Some:  option.WrapResult(nonsense(x.Unwrap())),
		option.Error: "ERROR",
	})
	//
	t.Logf("s = %+v, err = %v", s, err)
	if err != nil {
		t.Errorf("expected error from matching SomeInt(1) to be caught, isn't")
	}
	if s == nil {
		t.Errorf("expected int64(1) to match to a non-nil result, didn't")
	}
	if str, ok := s.(string); ok {
		if str != "ERROR" {
			t.Errorf("expected int64(1) to match to string ERROR, didn't")
		}
	} else {
		t.Errorf("expected int64(1) to match to string, didn't")
	}
}

// ---------------------------------------------------------------------------

func nonsense(x interface{}) (interface{}, error) {
	return nil, errors.New("ERROR")
}

func stringify(x interface{}) (interface{}, error) {
	return fmt.Sprintf("Value = %v", x), nil
}
