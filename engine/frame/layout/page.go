package layout

import (
	"container/heap"
	"sync"

	"github.com/npillmayer/tyse/core/dimen"
)

type Page struct {
	dimen.Rect             // page size
	queue      *EventQ     // every page manages an event queue (e.g., reflow events)
	template   interface{} // TODO
}

func NewPage(papersize dimen.Point) *Page {
	page := &Page{}
	page.Rect.BotR = papersize
	return page
}

// --- Event Queue -----------------------------------------------------------

type Event struct {
	priority uint8       // priority of the item in the queue.
	etype    EventType   // event type
	body     interface{} // arbitrary event body, depending on event type
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // index of the item in the heap.
}

type EventType uint8

const (
	VoidEvent EventType = iota
	ReflowEvent
	AbortEvent
)

func NewEvent(prio uint8, etype EventType) *Event {
	return &Event{
		priority: prio,
		etype:    etype,
	}
}

// A EventQ implements heap.Interface and holds events.
type EventQ struct {
	mutex  *sync.Mutex
	events []*Event
}

// Len is part of interface container/heap.
func (q EventQ) Len() int { return len(q.events) }

// Less is part of interface container/heap.
func (q EventQ) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return q.events[i].priority > q.events[j].priority
}

// Swap is part of interface container/heap.
func (q EventQ) Swap(i, j int) {
	q.events[i], q.events[j] = q.events[j], q.events[i]
	q.events[i].index = i
	q.events[j].index = j
}

// PushEvent pushes an event onto the queue.
func (q *EventQ) PushEvent(e *Event) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.Push(e)
}

// PopEvent pops an event from the head of the queue.
func (q *EventQ) PopEvent() *Event {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.Pop().(*Event)
}

// Update modifies the priority of an event in the queue.
func (q *EventQ) Update(e *Event, prio uint8) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	e.priority = prio
	heap.Fix(q, e.index)
}

// Push is part of interface container/heap.
// Not intended for client use.
func (q *EventQ) Push(x interface{}) {
	n := len(q.events)
	e := x.(*Event)
	e.index = n
	q.events = append(q.events, e)
}

// Pop is part of interface container/heap.
// Not intended for client use.
func (q *EventQ) Pop() interface{} {
	old := q.events
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	q.events = old[0 : n-1]
	return item
}
