package cache

import (
	"sync"

	"github.com/pkg/errors"
)

// Throttle is
type Throttle struct {
	loader  Loader
	m       map[string]Cache
	th      []string
	mutex   sync.Mutex
	sweeper *Sweeper
}

// NewThrottle is
func NewThrottle(loader Loader, capacity int, sweeper *Sweeper) *Throttle {
	return &Throttle{
		loader:  loader,
		m:       make(map[string]Cache),
		th:      make([]string, 0, capacity),
		sweeper: sweeper,
	}
}

// Get is
func (t *Throttle) Get(key string) (interface{}, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	c, ok := t.m[key]
	if !ok {
		newCache, err := t.loader(key)
		if err != nil {
			return nil, err
		}
		if t.sweeper != nil {
			newCache = t.sweeper.touch(newCache)
		}
		t.m[key] = newCache
		t.shift(key)
		err = newCache.Reload()
		if err != nil {
			return newCache.Get(), nil
		}
		return nil, err
	}

	updated, err := c.Updated()
	if err != nil {
		t.terminate(key)
		return nil, errors.Wrap(err, "Throttle.Get")
	} else if updated {
		err := c.Reload()
		if err != nil {
			t.terminate(key)
			return nil, errors.Wrap(err, "Throttle.Get")
		}
	}
	t.shift(key)
	return c.Get(), nil
}

func (t *Throttle) shift(key string) {
	// start index of shifting
	pos := -1
	l := len(t.th)

	// search
	for i := range t.th {
		if t.th[i] == key {
			pos = i
			break
		}
	}

	// when not found
	if pos == -1 {
		if l == cap(t.th) {
			pos = l - 1
			lastKey := t.th[pos]
			//delete(t.m, lastKey)
			t.terminate(lastKey)
		} else {
			pos = l
			t.th = append(t.th, "")
		}
	}
	for i := pos; i >= 0; i-- {
		if i == 0 {
			t.th[0] = key
			break
		}
		t.th[i] = t.th[i-1]
	}
}

// Terminate is
func (t *Throttle) Terminate() { // ============================================================================================-
	if t.sweeper != nil {
		close(t.sweeper.stop)
	}
	for _, c := range t.m {
		c.Release()
	}
	t.th = nil
}

func (t *Throttle) terminate(key string) {
	t.m[key].Release()
	delete(t.m, key)
}
