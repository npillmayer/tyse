package frame

import (
	"sync"
)

type FloatList struct {
	mutex  *sync.Mutex
	floats []Container
}

func (l *FloatList) AppendFloat(float Container) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.floats = append(l.floats, float)
}

func (l *FloatList) Contains(float Container) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, f := range l.floats {
		if f == float {
			return true
		}
	}
	return false
}

func (l *FloatList) Remove(float Container) bool {
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

func (l *FloatList) Floats() []Container {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	floats := make([]Container, len(l.floats))
	copy(floats, l.floats)
	return floats
}
