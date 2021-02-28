package option

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// Tracer traces to the core tracer.
func Tracer() tracing.Trace {
	return gtrace.CoreTracer
}
