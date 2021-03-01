/*
Package option implements an experimental option type.

Handling the infinite number of complicated rules in HTML and CSS I tend
to miss option types and matching. Without generics the implementation
in this package unfortunately is quite hacky and all-in-all defeats the
purpose of Go to be a simple, typesafe language. However, for now I'll prefer
a concise notation to type safety in some cases related to CSS styling and
layout. Let's see where it leads us.

As soon as generics will be part of Go, I'll rewrite this package (However,
I suspect someone else will quickly come up with an implementation of an option type,
or it even will find its way into the standard lib).
Clients probably should not use this package for now, except for experimental purposes.

The Int64T type is kind of a blueprint for other implementations of optional types,
i.e. types implementing the option.Type interface.

	x := option.SomeInt64(42)

	msg, _ = x.Match(option.Maybe{
		option.None: "attention: x is unset",
		option.Some: strcnv.Itoa(x.Unwrap(), 10),
	})

will yield a `msg` of

    "42"

whereas an unset option will be matching `option.None`:

    x := option.Int64()
    x.IsNone()  // => true

    "attention: x is unset"

To match concrete values, use `option.Of{…}`

	msg, _ = x.Match(option.Of{
		option.None: "no answer",
		42:          "best answer",
		…
		option.Some: "sub-par answer",
	}
__________________________________________________________________________

BSD 3-Clause License

Copyright (c) 2020–21, Norbert Pillmayer
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation
and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package option

import (
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// Tracer traces to the core tracer.
func Tracer() tracing.Trace {
	return gtrace.CoreTracer
}
