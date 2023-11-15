package tree

/*
License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2021 Norbert Pillmayer <norbert@pillmayer.com>

*/

import (
	"errors"
	"sync"
)

// ErrInvalidFilter is thrown if a pipeline filter step is defunct.
var ErrInvalidFilter = errors.New("filter stage is invalid")

// ErrEmptyTree is thrown if a Walker is called with an empty tree. Refer to
// the documentation of NewWalker() for details about this scenario.
var ErrEmptyTree = errors.New("cannot walk empty tree")

// ErrNoMoreFiltersAccepted is thrown if a client already called Promise(), but tried to
// re-use a walker with another filter.
var ErrNoMoreFiltersAccepted = errors.New("in promise mode; will not accept new filters; use a new walker")

// Walker holds information for operating on trees: finding nodes and
// doing work on them. Clients usually create a Walker for a (sub-)tree
// to search for a selection of nodes matching certain criteria, and
// then perform some operation on this selection.
//
// A Walker will eventually return two client-level values:
// A slice of tree nodes and the last error occured.
// Often these fields are accessed through a
// Promise-object, which represents future values for the two fields.
//
// A typical usage of a Walker looks like this ("FindNodesAndDoSomething()" is
// a placeholder for a sequence of function calls, see below):
//
//    w := NewWalker(node)
//    futureResult := w.FindNodesAndDoSomething(...).Promise()
//    nodes, err := futureResult()
//
// Walker support a set of search & filter functions. Clients will chain
// some of these to perform tasks on tree nodes (see examples).
// You may think of the set of operations to form a small
// Domain Specific Language (DSL), similar in concept to JQuery.
//
// ATTENTION: Clients must call Promise() as the final link of the
// DSL expression chain, even if they do not expect the expression to
// return a non-empty set of nodes. Firstly, they need to check for errors,
// and secondly without fetching the (possibly empty) result set by calling
// the promise, the Walker may leak goroutines.
type Walker[S, T comparable] struct {
	*sync.Mutex
	initial   *Node[S]        // initial node of (sub-)tree
	pipe      *pipeline[S, T] // pipeline of filters to perform work on tree nodes.
	promising bool            // client has called Promise()
}

func cloneWalker[S, T, U comparable](w *Walker[S, T], pipe *pipeline[S, U]) *Walker[S, U] {
	nw := &Walker[S, U]{
		initial:   w.initial,
		pipe:      pipe,
		promising: w.promising,
	}
	nw.Mutex = w.Mutex
	return nw
}

// NewWalker creates a Walker for the initial node of a (sub-)tree.
// The first subsequent call to a node filter function will have this
// initial node as input.
//
// If initial is nil, NewWalker will return a nil-Walker, resulting
// in a NOP-pipeline of operations, resulting in an empty set of nodes
// and an error (ErrEmptyTree).
func NewWalker[T comparable](initial *Node[T]) *Walker[T, T] {
	mx := new(sync.Mutex)
	if initial == nil {
		return nil
	}
	tracer().Debugf("new tree-walker, initial node = %v", initial)
	w := &Walker[T, T]{initial: initial, pipe: newPipeline[T](nil)}
	w.Mutex = mx
	return w
}

// appendFilterForTask will create a new filter for a task and append
// that filter at the end of the pipeline. If processing has not
// been started yet, it will be started.
func appendFilterForTask[S, T, U comparable](w *Walker[S, T], task workerTask[T, U], udata interface{},
	buflen int) (*Walker[S, U], error) {
	//
	if w.promising {
		return nil, ErrNoMoreFiltersAccepted
	}
	newFilter := newFilter(task, udata, buflen)
	if w.pipe.empty() { // quick check, may be false positive when in if-block
		// now we know the new filter might be the first one
		w.startProcessing() // this will check again, and startup if pipe empty
	}
	//w.pipe.appendFilter(newFilter) // insert filter in running pipeline
	pipe := AppendFilter[S, T, U](w.pipe, newFilter) // insert filter in running pipeline
	var newW *Walker[S, U] = cloneWalker(w, pipe)
	return newW, nil
}

// startProcessing should be called as soon as the first filter is inserted
// into the pipeline. It will put the initial tree node onto the front input
// channel.
func (w *Walker[S, T]) startProcessing() {
	doStart := false
	tracer().Debugf("tree walker starts processing")
	w.pipe.state.mx.RLock()
	if w.pipe.empty() { // no processing up to now => start with initial node
		w.pipe.pushSync(w.initial, 0) // input is buffered, will return immediately
		doStart = true                // yes, we will have to start the pipeline
	}
	w.pipe.state.mx.RUnlock()
	if doStart { // ok to be outside mutex as other goroutines will check pipe.empty()
		w.pipe.startProcessing() // must be outside of mutex lock
	}
}

// Promise is a future synchronisation point.
// Walkers may decide to perform certain tasks asynchronously.
// Clients will not receive the resulting node list immediately, but
// rather get handed a Promise.
// Clients will then—any time after they received the Promise—call the
// Promise (which is of function type) to receive a slice of nodes and
// a possible error value. Calling the Promise will block until all
// concurrent operations on the tree nodes have finished, i.e. it
// is a synchronization point.
func (w *Walker[S, T]) Promise() func() ([]*Node[T], error) {
	if w == nil {
		// empty Walker => wrap nil set and an error
		return func() ([]*Node[T], error) {
			return nil, ErrEmptyTree
		}
	}
	// drain the result channel and the error channel
	w.promising = true // will block calls to establish new filters
	errch := w.pipe.state.errors
	results := w.pipe.results
	counter := &w.pipe.state.queuecount
	signal := make(chan struct{}, 1)
	var selection []*Node[T]
	var lasterror error
	go func() {
		defer close(signal)
		selection, lasterror = waitForCompletion(results, errch, counter)
	}()
	// TODO : sort results
	return func() ([]*Node[T], error) {
		<-signal
		return selection, lasterror
	}
}

// ----------------------------------------------------------------------

// Predicate is a function type to match against nodes of a tree.
// Is is used as an argument for various Walker functions to
// collect a selection of nodes.
// test is the node under test, node is the input node.
type Predicate[T comparable] func(test *Node[T], node *Node[T]) (match *Node[T], err error)

// Whatever is a predicate to match anything (see type Predicate).
// It is useful to match the first node in a given direction.
func Whatever[T comparable]() Predicate[T] {
	return func(test *Node[T], node *Node[T]) (*Node[T], error) {
		return test, nil
	}
}

// NodeIsLeaf is a predicate to match leafs of a tree.
func NodeIsLeaf[T comparable]() Predicate[T] {
	return func(test *Node[T], node *Node[T]) (match *Node[T], err error) {
		if test.ChildCount() == 0 {
			return test, nil
		}
		return nil, nil
	}
}

// TraverseAll is a predicate to match nothing (see type Predicate).
// It is useful to traverse a whole tree.
/*
var TraverseAll Predicate = func(*Node) (bool, error) {
	return false, nil
}
*/

// ----------------------------------------------------------------------

// Parent returns the parent node.
//
// If w is nil, Parent will return nil.
func (w *Walker[S, T]) Parent() *Walker[S, T] {
	if w == nil {
		return nil
	}
	newW, err := appendFilterForTask(w, parent[T], nil, 0)
	//if err := w.appendFilterForTask(parent[T], nil, 0); err != nil {
	if err != nil {
		tracer().Errorf(err.Error())
		panic(err)
	}
	return newW
}

// parent is a very simple filter task to retrieve the parent of a tree node.
// if the node is the tree root node, parent() will not produce a result.
func parent[T comparable](node *Node[T], isBuffered bool, udata userdata, push func(*Node[T], uint32),
	pushBuf func(*Node[T], interface{}, uint32)) error {
	//
	p := node.Parent()
	serial := udata.serial
	if p != nil {
		push(p, serial) // forward parent node to next pipeline stage
	}
	return nil
}

// AncestorWith finds an ancestor matching the given predicate.
// The search does not include the start node.
//
// If w is nil, AncestorWith will return nil.
func (w *Walker[S, T]) AncestorWith(predicate Predicate[T]) *Walker[S, T] {
	if w == nil {
		return nil
	}
	if predicate == nil {
		w.pipe.state.errors <- ErrInvalidFilter
		return w
	}
	newW, err := appendFilterForTask(w, ancestorWith[T], predicate, 0)
	//err := w.appendFilterForTask(ancestorWith[T], predicate, 0) // hook in this filter
	if err != nil {
		tracer().Errorf(err.Error())
		panic(err)
	}
	return newW
}

// ancestorWith searches iteratively for an ancestor node matching a predicate.
// node is at least the parent of the start node or nil.
func ancestorWith[T comparable](node *Node[T], isBuffered bool, udata userdata, push func(*Node[T], uint32),
	pushBuf func(*Node[T], interface{}, uint32)) error {
	//
	if node == nil {
		return nil
	}
	predicate := udata.filterlocal.(Predicate[T])
	anc := node.Parent()
	serial := udata.serial
	for anc != nil {
		matchedNode, err := predicate(anc, node)
		if err != nil {
			return err
		}
		if matchedNode != nil {
			push(matchedNode, serial) // put ancestor on output channel for next pipeline stage
			return nil
		}
		anc = anc.Parent()
	}
	return nil // no matching ancestor found, not an error
}

// DescendentsWith finds descendents matching a predicate.
// The search does not include the start node.
//
// If w is nil, DescendentsWith will return nil.
func (w *Walker[S, T]) DescendentsWith(predicate Predicate[T]) *Walker[S, T] {
	if w == nil {
		return nil
	}
	if predicate == nil {
		w.pipe.state.errors <- ErrInvalidFilter
		return w
	}
	//err := w.appendFilterForTask(descendentsWith[T], predicate, 5) // need a helper queue
	newW, err := appendFilterForTask(w, descendentsWith[T], predicate, 5)
	if err != nil { // this should never happen here
		tracer().Errorf(err.Error())
		panic(err) // for debugging as long as this is unstable
	}
	return newW
}

func descendentsWith[T comparable](node *Node[T], isBuffered bool, udata userdata, push func(*Node[T], uint32),
	pushBuf func(*Node[T], interface{}, uint32)) error {
	//
	if isBuffered {
		predicate := udata.filterlocal.(Predicate[T])
		matchedNode, err := predicate(node, nil) // currently no origin node availabe
		serial := udata.serial
		if serial == 0 {
			serial = node.Rank
		}
		tracer().Debugf("Predicate for node %s returned: %v, err=%v", node, matchedNode, err)
		if err != nil {
			return err // do not descend further
		}
		if matchedNode != nil {
			push(matchedNode, serial) // found one, put on output channel for next pipeline stage
		}
		revisitChildrenOf(node, serial, pushBuf)
	} else {
		serial := udata.serial
		revisitChildrenOf(node, serial, pushBuf)
	}
	return nil
}

func revisitChildrenOf[T comparable](node *Node[T], serial uint32, pushBuf func(*Node[T], interface{}, uint32)) {
	chcnt := node.ChildCount()
	for position := 0; position < chcnt; position++ {
		if ch, ok := node.Child(position); ok {
			pp := parentAndPosition[T]{node, position}
			chSerial := node.calcChildSerial(serial, ch, position)
			pushBuf(ch, pp, chSerial)
		}
	}
}

// TODO this is too simplistic
func (node *Node[T]) calcChildSerial(myserial uint32, ch *Node[T], position int) uint32 {
	r := myserial - 1
	for i := node.ChildCount() - 1; i > position; i-- {
		if child, ok := node.Child(i); ok {
			r -= child.Rank
		}
	}
	return r
}

// AllDescendents traverses all descendents.
// The traversal does not include the start node.
// This is just a wrapper around `w.DescendentsWith(Whatever)`.
//
// If w is nil, AllDescendents will return nil.
func (w *Walker[S, T]) AllDescendents() *Walker[S, T] {
	return w.DescendentsWith(Whatever[T]())
}

// Filter calls a client-provided function on each node of the selection.
// The user function should return the input node if it is accepted and
// nil otherwise.
//
// If w is nil, Filter will return nil.
func (w *Walker[S, T]) Filter(f Predicate[T]) *Walker[S, T] {
	//func (w *Walker) Filter(f func(*Node) (*Node, error)) *Walker {
	if w == nil {
		return nil
	}
	if f == nil {
		w.pipe.state.errors <- ErrInvalidFilter
		return w
	}
	//err := w.appendFilterForTask(clientFilter[T], f, 0) // hook in this filter
	newW, err := appendFilterForTask(w, clientFilter[T], f, 0)
	if err != nil {
		tracer().Errorf(err.Error())
		panic(err)
	}
	return newW
}

//func clientFilter(node *Node, isBuffered bool, udata userdata, push func(*Node, uint32),
func clientFilter[T comparable](node *Node[T], isBuffered bool, udata userdata, push func(*Node[T], uint32),
	pushBuf func(*Node[T], interface{}, uint32)) error {
	//
	userfunc := udata.filterlocal.(Predicate[T])
	serial := udata.serial
	n, err := userfunc(node, node)
	if n != nil && err != nil {
		push(n, serial) // forward filtered node to next pipeline stage
	}
	return err
}

// Action is a function type to operate on tree nodes.
// Resulting nodes will be pushed to the next pipeline stage, if
// no error occured.
type Action[T comparable] func(n *Node[T], parent *Node[T], position int) (*Node[T], error)

// TopDown traverses a tree starting at (and including) the root node.
// The traversal guarantees that parents are always processed before
// their children.
//
// If the action function returns an error for a node,
// descending the branch below this node is aborted.
//
// If w is nil, TopDown will return nil.
func (w *Walker[S, T]) TopDown(action Action[T]) *Walker[S, T] {
	if w == nil {
		return nil
	}
	if action == nil {
		w.pipe.state.errors <- ErrInvalidFilter
		return w
	}
	//err := w.appendFilterForTask(topDown[T], action, 5) // need a helper queue
	newW, err := appendFilterForTask(w, topDown[T], action, 5)
	if err != nil {
		tracer().Errorf(err.Error())
		panic(err) // TODO for debugging purposes until more mature
	}
	return newW
}

// ad-hoc container
type parentAndPosition[T comparable] struct {
	parent   *Node[T]
	position int
}

func topDown[T comparable](node *Node[T], isBuffered bool, udata userdata, push func(*Node[T], uint32),
	pushBuf func(*Node[T], interface{}, uint32)) error {
	//
	if isBuffered { // node was received from buffer queue
		action := udata.filterlocal.(Action[T])
		var parent *Node[T]
		var position int
		if udata.nodelocal != nil {
			parent = udata.nodelocal.(parentAndPosition[T]).parent
			position = udata.nodelocal.(parentAndPosition[T]).position
		}
		serial := udata.serial
		if serial == 0 {
			serial = node.Rank
		}
		result, err := action(node, parent, position)
		tracer().Debugf("Action for node %s returned: %v, err=%v", node, result, err)
		if err != nil {
			return err // do not descend further
		}
		if result != nil {
			push(result, serial) // result -> next pipeline stage
		}
		revisitChildrenOf(node, serial, pushBuf) // hand over node as parent
	} else {
		serial := udata.serial
		pushBuf(node, nil, serial) // simply move incoming nodes over to buffer queue
	}
	return nil
}

type bottomUpFilterData[T comparable] struct {
	action       Action[T]
	childrenDict *rankMap[T]
}

// BottomUp traverses a tree starting at (and including) all the current nodes.
// Usually clients will select all of the tree's leafs before calling *BottomUp*().
// The traversal guarantees that parents are not processed before
// all of their children.
//
// If the action function returns an error for a node,
// the parent is processed regardless.
//
// If w is nil, BottomUp will return nil.
func (w *Walker[S, T]) BottomUp(action Action[T]) *Walker[S, T] {
	if w == nil {
		return nil
	}
	if action == nil {
		w.pipe.state.errors <- ErrInvalidFilter
		return w
	}
	filterdata := &bottomUpFilterData[T]{
		action:       action,
		childrenDict: newRankMap[T](),
	}
	//err := w.appendFilterForTask(bottomUp[T], filterdata, 5) // need a helper queue
	newW, err := appendFilterForTask(w, bottomUp[T], filterdata, 5)
	if err != nil {
		tracer().Errorf(err.Error())
		panic(err) // TODO for debugging purposes until more mature
	}
	return newW
}

func bottomUp[T comparable](node *Node[T], isBuffered bool, udata userdata, push func(*Node[T], uint32),
	pushBuf func(*Node[T], interface{}, uint32)) error {
	//
	if node.ChildCount() > 0 { // check if all children have been processed
		var bUpFilterData *bottomUpFilterData[T]
		bUpFilterData = udata.filterlocal.(*bottomUpFilterData[T])
		tracer().Debugf("bottom up filter data = %v", bUpFilterData)
		childCounter := bUpFilterData.childrenDict
		if int(childCounter.Get(node)) < node.ChildCount() {
			return nil
		} // else drop this node until last child processed
	}
	serial := udata.serial
	if isBuffered { // node was received from buffer queue
		position := 0
		parent := node.Parent()
		if parent != nil {
			position = parent.IndexOfChild(node)
		}
		action := udata.filterlocal.(*bottomUpFilterData[T]).action
		resultNode, err := action(node, parent, position)
		if err == nil && resultNode != nil {
			push(resultNode, serial) // result node -> next pipeline stage
		}
		if parent != nil { // if this is not a root node
			childCounter := udata.filterlocal.(*bottomUpFilterData[T]).childrenDict
			childCounter.Inc(parent)       // signal that one more child is done (ie., this node)
			pushBuf(parent, udata, serial) // possibly continue processing with parent
		}
	} else {
		pushBuf(node, udata, serial) // move start nodes over to buffer queue
	}
	return nil
}

// CalcRank is an action for bottom-up processing. It Calculates the 'rank'-member
// for each node, meaning: the number of child-nodes + 1.
// The root node will hold the number of nodes in the entire tree.
// Leaf nodes will have a rank of 1.
func CalcRank[T comparable](n *Node[T], parent *Node[T], position int) (*Node[T], error) {
	//
	r := uint32(1)
	for i := 0; i < n.ChildCount(); i++ {
		ch, ok := n.Child(i)
		if ok {
			r += ch.Rank
		}
	}
	n.Rank = r
	return n, nil
}
