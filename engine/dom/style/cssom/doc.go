/*
Package cssom provides functionality for CSS styling.

Status

This is a very first draft. It is unstable and the API will change without
notice. Please be patient.

Overview

HTMLbook is the core DOM of our documents.
Background for this decision can be found under
https://www.balisage.net/Proceedings/vol10/print/Kleinfeld01/BalisageVol10-Kleinfeld01.html
and http://radar.oreilly.com/2013/09/html5-is-the-future-of-book-authorship.html
For an in-depth description of HTMLbook please refer to
https://oreillymedia.github.io/HTMLBook/.

We strive to separate content from presentation. In typesetting, this is
probably an impossible claim, but we'll try anyway. Presentation
is governed with CSS (Cascading Style Sheets). CSS uses a box model more
complex than TeX's, which is well described here:

   https://developer.mozilla.org/en-US/docs/Learn/CSS/Introduction_to_CSS/Box_model

If you think about it: a typesetter using the HTML/CSS box model is
effectively a browser with output type PDF.
Browsers are large and complex pieces of code, a fact that implies that
we should seek out where to reduce complexity.

A good explanation of styling may be found in

   https://hacks.mozilla.org/2017/08/inside-a-super-fast-css-engine-quantum-css-aka-stylo/

CSSOM is the "CSS Object Model", similar to the DOM for HTML.
There is not very much open source Go code around for supporting us
in implementing a styling engine, except the great work of
https://godoc.org/github.com/andybalholm/cascadia.
Therefore we will have to compromise
on many feature in order to complete this in a realistic time frame.

This package relies on just one non-standard external library: cascadia.
CSS handling is de-coupled by introducing appropriate interfaces
StyleSheet and Rule. Concrete implementations may be found in sub-packages
of package style.

Further to consider:

   https://godoc.org/github.com/ericchiang/css
   https://golanglibs.com/search?q=css+parser&sort=top
   https://www.mediaevent.de/xhtml/style.html

The styling component is difficult to document/describe without
diagrams. Think about documenting with https://github.com/robertkrimen/godocdown.

___________________________________________________________________________

License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>
*/
package cssom

import "github.com/npillmayer/schuko/tracing"

// tracer traces with key 'tyse.dom'.
func tracer() tracing.Trace {
	return tracing.Select("tyse.dom")
}
