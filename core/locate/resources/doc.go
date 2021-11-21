/*
Package resources resolves all kinds of resources for an application.

As resource loading may be a time-consuming task, some functions in this
package will work in an async/await fashion by returning a promise.
Functions named

   Resolve…(…)

will return a resource-specific promise type, which the client will call later
to receive the loaded resource. The call to the promise-function will then block
until loading has completed.

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/
package resources

import (
	"github.com/npillmayer/schuko/tracing"
)

// tracer traces to tracing key 'tyse.resources'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.resources")
}
