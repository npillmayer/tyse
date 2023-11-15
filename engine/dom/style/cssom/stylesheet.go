package cssom

import "github.com/npillmayer/tyse/engine/dom/style"

// StyleSheet is an interface to abstract away a stylesheet-implementation.
// In order to de-couple implementations of CSS-stylesheets from the
// construction of the styled node tree, we introduce an interface
// for CSS stylesheets. Clients for the styling engine will have to
// provide a concrete implementation of this interface (e.g., see
// package douceuradapter).
//
// Having this interface imposes a performance hit. However, this
// implementation of CSS-styling will never trade modularity and
// clarity for performance. Clients in need for a production grade
// browser engine (where performance is key) should opt for headless
// versions of the main browser projects.
//
// See interface Rule.
type StyleSheet interface {
	AppendRules(StyleSheet) // append rules from another stylesheet
	Empty() bool            // does this stylesheet contain any rules?
	Rules() []Rule          // all the rules of a stylesheet
}

// Rule is the type stylesheets consists of.
//
// See interface StyleSheet.
type Rule interface {
	Selector() string            // the prelude / selectors of the rule
	Properties() []string        // property keys, e.g. "margin-top"
	Value(string) style.Property // property value for key, e.g. "15px"
	IsImportant(string) bool     // is property key marked as important?
}
