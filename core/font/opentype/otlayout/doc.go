/*
Package otlayout provides access to OpenType font layout features.

# Status

Work in progress.

# License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © Norbert Pillmayer <norbert@pillmayer.com>
*/
package otlayout

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/tyse/core"
)

// trace traces to a global core-tracer.
func trace() tracing.Trace {
	return gtrace.CoreTracer
}

// errFontFormat produces user level errors for font parsing.
func errFontFormat(x string) error {
	return core.Error(core.EINVALID, "OpenType font format: %s", x)
}
