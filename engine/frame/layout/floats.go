package layout

import (
	"sync"

	"github.com/npillmayer/tyse/engine/frame"
)

type FloatList struct {
	mutex  *sync.Mutex
	floats []frame.Container
}

func (l *FloatList) AppendFloat(float frame.Container) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.floats = append(l.floats, float)
}

func (l *FloatList) Contains(float frame.Container) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, f := range l.floats {
		if f == float {
			return true
		}
	}
	return false
}

func (l *FloatList) Remove(float frame.Container) bool {
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

func (l *FloatList) Floats() []frame.Container {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	floats := make([]frame.Container, len(l.floats))
	copy(floats, l.floats)
	return floats
}
