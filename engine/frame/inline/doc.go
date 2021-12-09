/*
Package inline produces line boxes from khipus.

_________________________________________________________________________

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package inline

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'tyse.frame'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.frame")
}
