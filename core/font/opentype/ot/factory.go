package ot

import "fmt"

// Navigator is an interface type to wrap various kinds of OpenType structure.
// On any given Navigator item, not all of the functions may result in sensible
// values returned. For example, OpenType map-like structures will return a
// map with a call to `Map`, but will return an invalid `Link` and an empty
// `List`. A Navigator may contain more than one OT structure, thus more than
// one such call may return a valid non-void entry.
//
// If a previous call in a navigation chain has caused an error, successing Navigator
// items will remember that error (call to `Error`) and will wrap only void Navigators
// (nil-safe).
type Navigator interface {
	Name() string  // returns the name of the underlying OpenType table
	Link() NavLink // non-void if Navigator contains a link
	Map() NavMap   // non-void if Navigator contains a map-like
	List() NavList // non-void if Navigator contains a list-like
	IsVoid() bool  // may be true if a previous error in the call chain occured
	Error() error  // previous error in call chain, if any
}

// NavList represents a sequence of—possibly unequal sized—items, addressable by
// position.
type NavList interface {
	Name() string        // returns the name of the underlying OpenType table
	Len() int            // number of items in the list
	Get(int) NavLocation // bytes of entry #n
	All() []NavLocation  // all entries as (possibly variable sized) byte segments
}

// NavMap wraps OpenType structures which are map-like. Lookup is always done on
// 32-bit values, even if the map's keys are 16-bit (will be shortened to low
// bytes in such cases).
//
// TagRecordMap is a special kind of NavMap.
type NavMap interface {
	Name() string
	Lookup(uint32) NavLocation
	LookupTag(Tag) NavLink
	IsTagRecordMap() bool
	AsTagRecordMap() TagRecordMap
}

// A TagRecordMap is a dict-type (map) to receive a data record (returned as a link)
// from a given tag. This kind of map is used within OpenType fonts in several
// instances, e.g.
// https://docs.microsoft.com/en-us/typography/opentype/spec/base#basescriptlist-table
//
// For some record maps the (tag) keys are not unique (e.g., the feature-list table),
// so in this case the first matching entry will be returned.
type TagRecordMap interface {
	Name() string           // OpenType specification name of this map
	LookupTag(Tag) NavLink  // returns the link associated with a given tag
	Tags() []Tag            // returns all the tags which the map uses as keys
	Len() int               // number of entries in the map
	Get(int) (Tag, NavLink) // get entry at position n
}

// NavigatorFactory creates a Navigator for a given OpenType object `obj` at location
// `loc`.
func NavigatorFactory(obj string, loc NavLocation, base NavLocation) Navigator {
	tracer().Debugf("navigator factory for %s", obj)
	switch obj {
	case "ScriptList":
		scriptRecords := parseTagRecordMap16(loc.Bytes(), 0, loc.Bytes(), "ScriptList", "Script")
		return linkAndMap{
			tmap: scriptRecords,
		}
	case "Script":
		l, err := parseLink16(loc.Bytes(), 0, loc.Bytes(), "LangSys")
		if err != nil {
			//return null(err)
			l = nullLink("no default script->langsys link")
		}
		tracer().Debugf("script table default langsys entry: %s", l.Name())
		return linkAndMap{
			link: l,
			tmap: parseTagRecordMap16(loc.Bytes(), 2, loc.Bytes(), "Script", "LangSys"),
		}
	case "LangSys":
		tracer().Debugf("%s[0] = %x", obj, u16(loc.Bytes()))
		tracer().Debugf("%s[2] = %x", obj, u16(loc.Bytes()[2:]))
		lsys, err := parseLangSys(loc.Bytes(), 2, "Feature-Index")
		if err != nil {
			return null(err)
		}
		return lsys
	case "Feature":
		l, err := parseLink16(loc.Bytes(), 0, loc.Bytes(), "Feature-Params")
		if err != nil {
			return null(err)
		}
		lookups, err := parseArray16(loc.Bytes(), 2, "Feature", "Feature-Lookups")
		if err != nil {
			return null(err)
		}
		return feature{
			params:  l,
			lookups: lookups,
		}
	case "name":
		names, err := parseNames(loc.Bytes())
		if err != nil {
			return null(err)
		}
		return names
	case "NameRecord":
		name, err := decodeUtf16(loc.Bytes())
		if err != nil {
			return null(err)
		}
		return navName{name: name}
	}
	if fields, ok := tableFields[obj]; ok {
		tracer().Debugf("object %s has fields %v", obj, fields)
		size := int(fields[0]) // total byte size of fields
		f := otFields{pattern: fields[1:], b: base.Bytes()[:size]}
		return list{navName: navName{name: obj}, f: f}
	}
	tracer().Debugf("no navigator found -> null navigator")
	return null(errDanglingLink(obj))
}

// The following code is work in progress -- expect it to change any second.

var tableFields = map[string][]uint8{
	// sum of fields is first entry
	"head": {54, 2, 2, 4, 4, 4, 2, 2, 8, 8, 2, 2, 2, 2, 2, 2, 2, 2, 2},
	"bhea": {54, 2, 2, 4, 4, 4, 2, 2, 8, 8, 2, 2, 2, 2, 2, 2, 2, 2, 2},
	"OS/2": {53, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 10, 4, 1, 2, 2, 2},
}

type navBase struct {
	err error
}

func (nbase navBase) Link() NavLink {
	return nullLink("generic link class")
}

func (nbase navBase) Map() NavMap {
	return tagRecordMap16{}
}

func (nbase navBase) List() NavList {
	return nullList
}

func (nbase navBase) IsVoid() bool {
	return true
}

func (nbase navBase) Name() string {
	if nbase.err != nil {
		return nbase.err.Error()
	}
	return "base nav: should not be visible"
}

func (nbase navBase) Error() error {
	return nbase.err
}

type linkAndMap struct {
	err  error
	link NavLink
	tmap NavMap
}

func (lm linkAndMap) Link() NavLink {
	return lm.link
}

func (lm linkAndMap) Map() NavMap {
	return lm.tmap
}

func (lm linkAndMap) List() NavList {
	return nullList
}

func (lm linkAndMap) IsVoid() bool {
	return lm.tmap == nil
}

func (lm linkAndMap) Name() string {
	return lm.tmap.Name()
}

func (lm linkAndMap) Error() error {
	return lm.err
}

func null(err error) Navigator {
	return navBase{err: err}
}

func nullLink(errmsg string) NavLink {
	return link16{err: fmt.Errorf("link: %s", errmsg)}
}

func errDanglingLink(obj string) error {
	return fmt.Errorf("cannot resolve link to %s", obj)
}

// var nullNav = linkAndMap{}
var nullList = u16List{}

type navName struct {
	navBase
	name string
}

func (nm navName) Name() string {
	return nm.name
}

type list struct {
	navName
	f otFields
}

func (l list) IsVoid() bool {
	return false
}

func (l list) List() NavList {
	return l.f
}

type otFields struct {
	name    string
	pattern []uint8
	b       binarySegm
}

func (f otFields) Name() string {
	return "fields"
}

func (f otFields) Len() int {
	return len(f.pattern)
}

func (f otFields) Get(i int) NavLocation {
	if i < 0 || i >= len(f.pattern) {
		return binarySegm{}
	}
	offset := 0
	for j, p := range f.pattern {
		if j > i {
			break
		}
		offset += int(p)
	}
	if r, err := f.b.view(offset, int(f.pattern[i])); err == nil {
		return r
	}
	return binarySegm{}
}

func (f otFields) All() []NavLocation {
	r := make([]NavLocation, 0, len(f.pattern))
	offset := 0
	for _, p := range f.pattern {
		if x, err := f.b.view(offset, int(p)); err == nil {
			r = append(r, x)
		} else {
			return []NavLocation{binarySegm{}}
		}
	}
	return r
}
