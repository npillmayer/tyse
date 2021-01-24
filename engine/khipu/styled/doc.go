// Package styled creates styled paragraphs from a W3CDOM.
package styled

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// T traces to a global engine-tracer.
func T() tracing.Trace {
	return gtrace.EngineTracer
}
