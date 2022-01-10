package frame

import (
	"sync"
)

type FloatList struct {
	mutex  *sync.Mutex
	floats []ContainerInterf
}

func (l *FloatList) AppendFloat(float ContainerInterf) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.floats = append(l.floats, float)
}

func (l *FloatList) Contains(float ContainerInterf) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, f := range l.floats {
		if f == float {
			return true
		}
	}
	return false
}

func (l *FloatList) Remove(float ContainerInterf) bool {
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

func (l *FloatList) Floats() []ContainerInterf {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	floats := make([]ContainerInterf, len(l.floats))
	copy(floats, l.floats)
	return floats
}
