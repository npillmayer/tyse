package layout

import (
	"sync"

	"github.com/npillmayer/tyse/engine/frame/boxtree"
)

type FloatList struct {
	mutex  *sync.Mutex
	floats []boxtree.Container
}

func (l *FloatList) AppendFloat(float boxtree.Container) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.floats = append(l.floats, float)
}

func (l *FloatList) Contains(float boxtree.Container) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, f := range l.floats {
		if f == float {
			return true
		}
	}
	return false
}

func (l *FloatList) Remove(float boxtree.Container) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for i, f := range l.floats {
		if f == float {
			l.floats = append(l.floats[:i], l.floats[i+1:]...)
			return true
		}
	}
	return false
}

func (l *FloatList) Floats() []boxtree.Container {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	floats := make([]boxtree.Container, len(l.floats))
	copy(floats, l.floats)
	return floats
}
