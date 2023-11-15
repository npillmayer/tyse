package tree

import (
	"fmt"
	"sync"
)

var errRankOfNullNode = fmt.Errorf("cannot determine rank of null-node")

type rankMap[T comparable] struct {
	lock  *sync.RWMutex
	count map[*Node[T]]uint32
}

func newRankMap[T comparable]() *rankMap[T] {
	return &rankMap[T]{
		&sync.RWMutex{},
		make(map[*Node[T]]uint32),
	}
}

func (rmap *rankMap[T]) Put(n *Node[T], r uint32) (uint32, error) {
	if n == nil {
		return 0, errRankOfNullNode
	}
	if rmap == nil {
		rmap = newRankMap[T]()
	}
	rmap.lock.RLock()
	rank := rmap.count[n]
	rmap.lock.RUnlock()
	rmap.lock.Lock() // race condition
	defer rmap.lock.Unlock()
	rmap.count[n] = r
	return rank, nil
}

func (rmap *rankMap[T]) Get(n *Node[T]) uint32 {
	if n == nil || rmap == nil {
		return 0
	}
	rmap.lock.RLock()
	defer rmap.lock.RUnlock()
	rank := rmap.count[n]
	return rank
}

func (rmap *rankMap[T]) Inc(n *Node[T]) (uint32, error) {
	if n == nil {
		return 0, errRankOfNullNode
	}
	if rmap == nil {
		rmap = newRankMap[T]()
	}
	rmap.lock.Lock()
	defer rmap.lock.Unlock()
	rank := rmap.count[n]
	rmap.count[n] = rank + 1
	return rank, nil
}

func (rmap *rankMap[T]) Clear(n *Node[T]) uint32 {
	if n == nil || rmap == nil {
		return 0
	}
	rmap.lock.Lock()
	defer rmap.lock.Unlock()
	rank := rmap.count[n]
	delete(rmap.count, n)
	return rank
}

// --------------------------------------------------------------------------------

// a helper struct for ordering the resulting nodes and their serials
type resultSlices[T comparable] struct {
	nodes   []*Node[T]
	serials []uint32
}

func (rs resultSlices[T]) Len() int           { return len(rs.nodes) }
func (rs resultSlices[T]) Less(i, j int) bool { return rs.serials[i] < rs.serials[j] }
func (rs resultSlices[T]) Swap(i, j int) {
	rs.nodes[i], rs.nodes[j] = rs.nodes[j], rs.nodes[i]
	rs.serials[i], rs.serials[j] = rs.serials[j], rs.serials[i]
}
