/*
Package css provides functionality for CSS styling.

CSS properties are plentyful and some of them are complicated.
This package trys to shield clients from the cumbersome handling of
CSS properties resulting of (1) the textual nature of CSS properties
and (2) the complicated semantics of computing style attributes for a
given node.

Status

This is a very first draft. It is unstable and the API will change without
notice. Please be patient.


License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>

*/
package css

// see
// https://developer.mozilla.org/en-US/docs/Web/CSS/Reference#dom-css_cssom

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces with key 'tyse.dom'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.dom")
}
