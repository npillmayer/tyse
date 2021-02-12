package layout

import (
	"sync"

	"github.com/npillmayer/tyse/engine/dom"
	"github.com/npillmayer/tyse/engine/frame"
	"golang.org/x/net/html"
)

type domToBoxAssoc struct {
	sync.RWMutex
	m map[*html.Node]frame.Container
}

func newAssoc() *domToBoxAssoc {
	return &domToBoxAssoc{
		m: make(map[*html.Node]frame.Container),
	}
}

func (d2c *domToBoxAssoc) Put(domnode *dom.W3CNode, c frame.Container) {
	d2c.Lock()
	defer d2c.Unlock()
	d2c.m[domnode.HTMLNode()] = c
}

func (d2c *domToBoxAssoc) Get(domnode *dom.W3CNode) (frame.Container, bool) {
	d2c.RLock()
	defer d2c.RUnlock()
	c, ok := d2c.m[domnode.HTMLNode()]
	return c, ok
}

func (d2c *domToBoxAssoc) Length() int {
	d2c.RLock()
	defer d2c.RUnlock()
	return len(d2c.m)
}
