package tree

/*
License

Governed by a 3-Clause BSD license. License file may be found in the root
folder of this module.

Copyright © 2017–2022 Norbert Pillmayer <norbert@pillmayer.com>

*/

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
)

// Tree operations will be carried out by concurrent worker goroutines.
// As tree operations may be chained, a pipeline of filter stages is
// constructed. Every chained operation is reflected by a filter stage.
// Filters read Nodes from an input channel and put processed Nodes on
// an output channel. This way we create a little pipes&filter design.
//
// Filter stages operate concurrently. Every filter is free to launch
// as many worker goroutines as it sees fit. An overall counter is used
// to track the number of active work-packages (i.e. Nodes) in the
// pipeline. As soon as the number of nodes is zero, all channels (pipes)
// are closed and the workers will terminate.
//
// Every filter performs a specific task, reflected by a workerTask function.
// Filter tasks may use additional data, which may be provided as an
// untyped udata ("user data") argument. Filter task functions are responsible
// for decoding their specific udata.
// Errors occuring in filter tasks will be sent to a pipeline-global error
// channel.

// Minimum and maximum number of concurrent workers for a tree operation
// (filter).
const (
	minWorkerCount int = 3
	maxWorkerCount int = 10
)

// Maxmimum length of internal buffer channel for a filter.
const maxBufferLength int = 128

// Workers will be tasked a series of workerTasks.
//
// node: input tree node
// isbuffered: is the input node from this stage's buffer queue?
// udata: user provided additional data
// emit:  // function to emit result node to next stage
// buffer: function to queue node in local buffer
//
// Does not return anything except a possible error condition.
type workerTask[S, T comparable] func(
	node *Node[S],
	isbuffered bool,
	udata userdata,
	emit func(*Node[T], uint32),
	buffer func(*Node[S], interface{}, uint32)) error

type stage interface {
	Shutdown()
}

// filter is part of a pipeline, i.e. a stage of the overall pipeline to
// process input (Nodes) and produce results (Nodes).
// filters will perform concurrently.
type filter[S, T comparable] struct {
	results    chan<- nodePackage[T] // results of this filter (pipeline stage)
	queue      chan nodePackage[S]   // helper queue if necessary
	task       workerTask[S, T]      // the task this filter performs
	filterdata interface{}           // user-provided information needed to perform task
	env        *filterenv[S]         // connection to outside world
}

func (f *filter[S, T]) Shutdown() {
	close(f.results)
	if f.queue != nil {
		close(f.queue)
	}
}

// nodePackage is the type which is transported in a pipeline.
// Each pipeline stage emits an instance of this type to the next stage.
//
// 'nodelocal' lets clients store arbitrary user data together with the node.
// It will be set to 'nil' as soon as the nodepackage is transferred to the next stage,
// i.e., this type is local to a pipeline-stage/filter.
type nodePackage[T comparable] struct {
	node      *Node[T]    // tree node
	nodelocal interface{} // arbitrary user data
	serial    uint32      // serial number of node for ordering
}

// filterenv holds information about the outside world to be referenced by
// a filter. This includes input workload, error destination and a counter
// for overall work on an pipeline.
type filterenv[T comparable] struct {
	input        <-chan nodePackage[T] // work to do for this filter, connected to predecessor
	errors       chan<- error          // where errors are reported to
	queuecounter *sync.WaitGroup       // counter for overall work load
}

// userdata is a container managed by the pipeline mechanism. It will contain
// two types of data availble for filters:
// ■ private information local to a filter (filterlocal),
// ■ and information acompanying a single node (nodelocal & serial).
// The pipeline mechanism will construct this from the filter environment and from
// node-local user-managed data, and it will deconstruct it for calls to a 'task()'.
type userdata struct {
	filterlocal interface{}
	nodelocal   interface{}
	serial      uint32
}

// newFilter creates a new pipeline stage, i.e. a filter fed from an input
// channel (workload). the filter is expected to put processed nodes into an
// output channel (results).
//
// Errors are reported to an error channel.
func newFilter[S, T comparable](task workerTask[S, T], filterdata interface{}, buflen int) *filter[S, T] {
	f := &filter[S, T]{}
	if buflen > 0 {
		if buflen > maxBufferLength {
			buflen = maxBufferLength
		}
		f.queue = make(chan nodePackage[S], buflen)
	}
	f.task = task
	f.filterdata = filterdata
	return f
}

// This method signature is a bit strange, but for now it does the job.
// Sets an environment for a filter an gets the results-channel in return.
func (f *filter[S, T]) start(env *filterenv[S]) chan nodePackage[T] {
	f.env = env
	res := make(chan nodePackage[T], 3) // output channel has to be in place before workers start
	f.results = res                     // be careful to set write-only for the filter
	n := runtime.NumCPU()
	if n > maxWorkerCount {
		n = maxWorkerCount
	} else if n < minWorkerCount {
		n = minWorkerCount
	}
	for i := 0; i < n; i++ {
		wno := i + 1
		if f.queue == nil {
			go filterWorker(f, wno) // startup worker no. #wno
		} else {
			go filterWorkerWithQueue(f, wno) // startup worker no. #wno
		}
	}
	return res // needed r/w for next filter in pipe
}

// filterWorker is the default worker function. Each filter is free to start
// as many of them as seems adequate.
//
// Each worker is identified through a worker number 'wno'.
func filterWorker[S, T comparable](f *filter[S, T], wno int) {
	//  defer func() {
	//	log.Printf("finished worker #%d\n", wno) // for debugging
	//}()
	push := func(node *Node[T], serial uint32) { // worker will use this to hand result to next stage
		f.pushResult(node, serial)
	}
	for inNode := range f.env.input { // get workpackages until drained
		node := inNode.node
		serial := inNode.serial
		udata := userdata{f.filterdata, nil, serial}
		err := f.task(node, false, udata, push, nil) // perform task on workpackage
		if err != nil {
			f.env.errors <- err // signal error to caller
		}
		qid := fmt.Sprintf("[#%p]", f.env.queuecounter)
		tracer().Debugf("filter stage %d finished -1 task for %v | %d in %s", wno, node, serial, qid)
		f.env.queuecounter.Done() // worker has finished a workpackage
	}
}

// filterWorkerWithQueue is a worker function which uses a separate support
// queue, the 'buffer queue'. This buffer queue may be used to re-schedule nodes
// until they are completely processed.
func filterWorkerWithQueue[S, T comparable](f *filter[S, T], wno int) {
	push := func(node *Node[T], serial uint32) { // worker will use this to hand result to next stage
		f.pushResult(node, serial)
	}
	pushBuf := func(sup *Node[S], udata interface{}, serial uint32) { // worker will use this to queue work internally
		f.pushBuffer(sup, udata, serial)
	}
	var buffered bool
	var node *Node[S]
	var udata userdata
	for {
		select { // get upstream workpackages and buffered workpackages until drained
		case inNode := <-f.env.input:
			node = inNode.node
			udata.serial = inNode.serial
			udata.nodelocal = nil
			udata.filterlocal = f.filterdata
			buffered = false
		case supdata := <-f.queue:
			node = supdata.node
			udata.filterlocal = f.filterdata
			udata.nodelocal = supdata.nodelocal
			udata.serial = supdata.serial
			buffered = true
		}
		if node != nil {
			err := f.task(node, buffered, udata, push, pushBuf) // perform filter task
			if err != nil {
				f.env.errors <- err // signal error to caller
			}
			qid := fmt.Sprintf("[#%p]", f.env.queuecounter)
			tracer().Debugf("filter stage %d finished -1 buffered task for %v | %d in %s", wno, node, udata.serial, qid)
			f.env.queuecounter.Done() // worker has finished a workpackage
		} else {
			break // no more work to do
		}
	}
}

// pipeline is a chain of filters to perform tasks on Nodes.
// Filters, i.e., pipeline stages are connected by channels.
type pipeline[S, T comparable] struct {
	input   chan nodePackage[S] // initial workload
	results chan nodePackage[T] // where final output of this pipeline goes to
	state   *pipelineState      // mutable state all incarnations of a pipeline refer to
}

// pipelineState is the mutable part of a pipeline, shared by all incarnations of a
// pipeline. This is necessary for synchronization.
type pipelineState struct {
	mx         sync.RWMutex   // to sychronize access to various fields
	queuecount sync.WaitGroup // overall count of work packages
	errors     chan error     // collector channel for error messages
	stages     []stage        // chain of stages/filters
	running    bool           // is this pipeline processing?
}

func newPipelineState() *pipelineState {
	state := &pipelineState{errors: make(chan error, 20)}
	return state
}

func (pstate *pipelineState) appendStage(s stage) {
	pstate.stages = append(pstate.stages, s)
}

// newPipeline creates an empty pipeline.
func newPipeline[T comparable](state *pipelineState) *pipeline[T, T] {
	pipe := &pipeline[T, T]{}
	if state == nil {
		state = newPipelineState()
	}
	pipe.state = state
	pipe.input = make(chan nodePackage[T], 10)
	pipe.results = pipe.input // short-curcuit, will be filled with filters
	return pipe
}

// clonePipeline creates a new pipeline from an existing one.
// It will not connect the queues, but it will create a new result queue of type U.
func clonePipeline[S, T, U comparable](p *pipeline[S, T]) *pipeline[S, U] {
	pipe := &pipeline[S, U]{state: p.state}
	pipe.input = p.input
	pipe.results = make(chan nodePackage[U], 10)
	return pipe
}

// Is this pipeline empty, i.e., has no filter stages yet?
func (pipe *pipeline[S, T]) empty() bool {
	pipe.state.mx.RLock()
	defer pipe.state.mx.RUnlock()
	return len(pipe.state.stages) == 0
}

// pushResult puts a node on the results channel of a filter stage (non-blocking).
// It is used by filter workers to communicate a result to the next stage
// of a pipeline.
func (f *filter[S, T]) pushResult(node *Node[T], serial uint32) {
	qid := fmt.Sprintf("[#%p]", f.env.queuecounter)
	tracer().Debugf("filter stage pushes +1 result %v | %d to %s", node, serial, qid)
	f.env.queuecounter.Add(1)
	written := true
	select { // try to send it synchronously without blocking
	case f.results <- nodePackage[T]{node, nil, serial}:
	default:
		written = false
	}
	if !written { // nope, we'll have to go async
		go func(node *Node[T], serial uint32) {
			f.results <- nodePackage[T]{node, nil, serial}
		}(node, serial)
	}
}

// pushBuffer puts a node on the buffer queue of a filter
// (non-blocking).
func (f *filter[S, T]) pushBuffer(node *Node[S], udata interface{}, serial uint32) {
	nodesup := nodePackage[S]{node, udata, serial}
	qid := fmt.Sprintf("[#%p]", f.env.queuecounter)
	tracer().Debugf("filter stage buffers +1 node %v | %d to %s", node, serial, qid)
	f.env.queuecounter.Add(1) // overall workload increases
	written := true
	select { // try to send it synchronously without blocking
	case f.queue <- nodesup:
	default:
		written = false
	}
	if !written { // nope, we'll have to go async
		go func(sup nodePackage[S]) {
			f.queue <- sup
		}(nodesup)
	}
}

// appendFilter appends a filter to a pipeline, i.e. as the last stage of
// the pipeline. Connects input- and output-channels appropriately and
// sets an environment for the filter.
func AppendFilter[S, T, U comparable](pipe *pipeline[S, T], f *filter[T, U]) *pipeline[S, U] {
	tracer().Debugf("append tree filter")
	pipe.state.mx.Lock()
	defer pipe.state.mx.Unlock()
	var newpipe *pipeline[S, U] = clonePipeline[S, T, U](pipe)
	newpipe.state.appendStage(f)
	tracer().Debugf("adding new stage/filter to pipeline, now #%d", len(newpipe.state.stages))
	env := &filterenv[T]{} // now set the environment for the filter
	env.errors = pipe.state.errors
	env.queuecounter = &pipe.state.queuecount
	env.input = pipe.results       // current output is input to new filter stage
	newpipe.results = f.start(env) // remember new final output
	return newpipe
}

// startProcessing starts a pipeline. It will start a watchdog goroutine
// to wait for the overall number of work packages to become zero.
// The watchdog will close all channels as soon as no more work
// packages (i.e., Nodes) are in the pipeline.
// Pre-requisite: at least one node/task in the front input channel.
//
// TODO pipe.stages is stale due to cloning of pipeline!
func (pipe *pipeline[S, T]) startProcessing() {
	pipe.state.mx.Lock()
	defer pipe.state.mx.Unlock()
	if !pipe.state.running {
		pipe.state.running = true
		go func() { // cleanup function
			qid := fmt.Sprintf("[%p]", &pipe.state.queuecount)
			tracer().Debugf("started waiting for empty node queue %s ...", qid)
			pipe.state.queuecount.Wait() // wait for empty queues
			tracer().Debugf("shutting down...")
			close(pipe.state.errors)
			close(pipe.input)
			tracer().Debugf("closed error- and input-pipes, now shutting down %d stages...", len(pipe.state.stages))
			for _, f := range pipe.state.stages {
				tracer().Debugf("shutting down filter")
				f.Shutdown()
			}
			pipe.state.running = false
		}()
	}
}

// pushSync synchronously puts a node on the input channel of a pipeline.
func (pipe *pipeline[S, T]) pushSync(node *Node[S], serial uint32) {
	qid := fmt.Sprintf("[#%p]", &pipe.state.queuecount)
	tracer().Debugf("pipeline sync start pushes +1 node %v | %d to %s", node, serial, qid)
	pipe.state.queuecount.Add(1)
	pipe.input <- nodePackage[S]{node, nil, serial} // input q is buffered
}

// pushAsync asynchronously puts a node on the input channel of a pipeline.
func (pipe *pipeline[S, T]) pushAsync(node *Node[S], serial uint32) {
	go func(node *Node[S]) {
		qid := fmt.Sprintf("[#%p]", &pipe.state.queuecount)
		tracer().Debugf("pipeline async start pushes +1 node %v | %d to %s", node, serial, qid)
		pipe.state.queuecount.Add(1)
		pipe.input <- nodePackage[S]{node, nil, serial} // input q is buffered
	}(node)
}

// waitForCompletion blocks until all work packages of a pipeline are done.
// It will receive the results of the final filter stage of the pipeline
// and collect them into a slice of Nodes. The slice will be a set, i.e.
// not contain duplicate Nodes.
func waitForCompletion[T comparable](results <-chan nodePackage[T], errch <-chan error, counter *sync.WaitGroup) ([]*Node[T], error) {
	// Collect all results from the pipeline
	var selection []*Node[T]       // slice of nodes -> return value
	var serials []uint32           // slice of serial numbers for ordering
	m := make(map[*Node[T]]uint32) // intermediate map to suppress duplicates
	for nodepkg := range results { // drain results channel
		m[nodepkg.node] = nodepkg.serial // remember last serial for node (may be random)
		qid := fmt.Sprintf("[#%p]", counter)
		tracer().Debugf("extracted -1 result from %s", qid)
		counter.Done() // we removed a value => count down
	}
	for node, serial := range m { // extract unique results into slices
		selection = append(selection, node) // collect unique return values
		serials = append(serials, serial)
		// resultSlices is a helper struct for sorting
		// it implements the Sort interface
		if len(selection) > 0 && selection[0].Rank > 0 { // if rank is unset: no sorting possible
			sort.Sort(resultSlices[T]{selection, serials})
		}
		// after this, serials are discarded
	}
	// Get last error from error channel
	var lasterror error
	for err := range errch {
		if err != nil {
			lasterror = err // throw away all errors but the last one
		}
	}
	return selection, lasterror
}
