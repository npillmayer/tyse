/*
Package monospace implements a simple shaper for monospace output.


License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package monospace

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'tyse.glyphs'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.glyphs")
}
