package option

import (
	"errors"
	"math"
	"strconv"
)

var ErrNoSuchMatchPattern = errors.New("no such match pattern")
var ErrCannotMatchUnsetValue = errors.New("cannot match unset value")
var ErrCannotMatchValue = errors.New("cannot match value")

type MaybeOption int

const (
	None MaybeOption = iota
	Some
	Error
)

// Maybe is a type used for matching of optional types.
// It will match `Some` if a value is set, `None` if it is unset, or `Error`
// if an error occurs.
type Maybe map[MaybeOption]interface{}

// Of is a type used for matching of optional types.
// It will first try to match concrete values, and in case of no match will
// then try a Maybe match.
type Of map[interface{}]interface{}

//type expr func(interface{}) func(interface{}, MaybeOption) interface{}

// Type is a type for optional values.
type Type interface {
	Match(choices interface{}) (interface{}, error)
	Equals(other interface{}) bool
	IsNone() bool
	//Expr(interface{}) expr
}

// Match will do a standard matching of o against choices.
// It may be used to create a new type of interface OptionT.
//
// choices are expected to be a map type, where keys of the map are either
// concrete values for o, or of type MaybeOption. Values of the map may be
// of any type.
//
// If choices is of unknown kind, nil and ErrNoSuchMatchPattern are returned.
//
func Match(o Type, choices interface{}) (value interface{}, err error) {
	switch c := choices.(type) {
	case Of:
		return c.Match(o)
	case Maybe:
		return c.Match(o)
	}
	return nil, ErrNoSuchMatchPattern
}

func (of Of) Match(o Type) (value interface{}, err error) {
	Tracer().Debugf("Match(Type=%T) for %T", of, o)
	if o.IsNone() {
		Tracer().Debugf("o is None")
		if expr, ok := of[None]; ok {
			Tracer().Debugf("matched nil expr=%T %v", expr, expr)
			value, err = valueOrExpr(expr, o, None)
		} else {
			err = ErrCannotMatchUnsetValue
		}
	} else {
		err = ErrCannotMatchValue
		for k, expr := range of {
			if o.Equals(k) {
				Tracer().Debugf("matched expr=%T %v", expr, expr)
				value, err = valueOrExpr(expr, o, Some)
			}
		}
		if err != nil {
			if expr, ok := of[Some]; ok {
				Tracer().Debugf("matched some expr=%T %v", expr, expr)
				value, err = valueOrExpr(expr, o, Some)
			}
			if err != nil {
				Tracer().Errorf(err.Error())
				if expr, ok := of[Error]; ok {
					value, err = valueOrExpr(expr, o, Error)
				}
			}
		}
	}
	Tracer().Debugf("===> return %v (%T) with error=%v", value, value, err)
	return value, err
}

func (maybe Maybe) Match(o Type) (value interface{}, err error) {
	Tracer().Debugf("Match(Type=%T) for %T", maybe, o)
	if o.IsNone() {
		Tracer().Debugf("o is None")
		if expr, ok := maybe[None]; ok {
			Tracer().Debugf("matched nil expr=%T %v", expr, expr)
			value, err = valueOrExpr(expr, o, None)
		} else {
			err = ErrCannotMatchUnsetValue
		}
	} else {
		if expr, ok := maybe[Some]; ok {
			Tracer().Debugf("matched some expr=%T %v", expr, expr)
			value, err = valueOrExpr(expr, o, Some)
		}
		if err != nil {
			Tracer().Errorf(err.Error())
			if expr, ok := maybe[Error]; ok {
				value, err = valueOrExpr(expr, o, Error)
			}
		}
	}
	Tracer().Debugf("===> return %v (%T) with error=%v", value, value, err)
	return value, err
}

func valueOrExpr(op interface{}, value Type, t MaybeOption) (interface{}, error) {
	Tracer().Debugf("value or expr %v(%v), t=%v", op, value, t)
	switch x := op.(type) {
	case func(interface{}, MaybeOption) (interface{}, error):
		Tracer().Debugf("calling func(value, type)")
		return x(value, t)
	case func(interface{}) (interface{}, error):
		Tracer().Debugf("calling func(value)")
		return x(value)
	}
	return op, nil
}

// --- Int64T-----------------------------------------------------------------

// Int64T is an option type for int64.
type Int64T int64

// Int64None is used as an in-band null value for type int64 for optional integers.
const Int64None int64 = math.MaxInt64

// SomeInt64 creates an optional int64 with an initial value of x.
func SomeInt64(x int) Int64T {
	return Int64T(x)
}

// Int64 creates an optional int64 without an initial value.
func Int64() Int64T {
	return Int64T(Int64None)
}

func (o Int64T) Match(choices interface{}) (value interface{}, err error) {
	return Match(o, choices)
}

func (o Int64T) Equals(other interface{}) bool {
	Tracer().Debugf("EQUALS %v ? %v", o, other)
	switch i := other.(type) {
	case int64:
		return int64(o) == i
	case int32:
		return int64(o) == int64(i)
	case int:
		return int64(o) == int64(i)
	}
	return false
}

func (o Int64T) Unwrap() int64 {
	return int64(o)
}

// IsNone returns true if o is unset.
func (o Int64T) IsNone() bool {
	return o == Int64T(Int64None)
}

func (o Int64T) String() string {
	if o.IsNone() {
		return "Int64.None"
	}
	return strconv.FormatInt(int64(o), 10)
}

// --- reference types -------------------------------------------------------

type RefT struct {
	ref interface{}
}

func (o RefT) Equals(other interface{}) bool {
	return o.ref == other
}

func (o RefT) IsNone() bool {
	return o.ref == nil
}

func (o RefT) Unwrap() interface{} {
	return o.ref
}

func Something(x interface{}) RefT {
	return RefT{ref: x}
}

func Nothing() RefT {
	return RefT{ref: nil}
}

func (o RefT) Match(choices interface{}) (value interface{}, err error) {
	return Match(o, choices)
}

var _ Type = RefT{}

// ---------------------------------------------------------------------------

// Safe wraps a Match's return values and drops the error value.
func Safe(x interface{}, err error) interface{} {
	return x
}
