package cache

import (
	"sync"
)

// Throttle is
type Throttle struct {
	factory Factory
	m       map[string]Cache
	th      []string
	mutex   sync.Mutex
	sweeper *Sweeper
}

// NewThrottle is
func NewThrottle(factory Factory, capacity int, sweeper *Sweeper) *Throttle {
	return &Throttle{
		factory: factory,
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
		newCache, err := t.factory(key)
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
			return nil, err
		}
		return newCache.Get(), nil
	}

	updated, err := c.Updated()
	if err != nil {
		t.terminate(key)
		return nil, &OpError{"update", err}
	} else if updated {
		err := c.Reload()
		if err != nil {
			t.terminate(key)
			return nil, &OpError{"reload", err}
		}
	}
	t.shift(key)
	return c.Get(), nil
}

// Reset is
func (t *Throttle) Reset(key string, v interface{}) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	c, ok := t.m[key]
	if !ok {
		var err error
		c, err = t.factory(key)
		if err != nil {
			return err
		}
		if t.sweeper != nil {
			c = t.sweeper.touch(c)
		}
		t.m[key] = c
	}
	t.shift(key)
	err := c.Reset(v)
	if err != nil {
		return err
	}
	return nil
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
func (t *Throttle) Terminate() {
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
	for i := range t.th {
		if t.th[i] == key {
			t.th = append(t.th[:i], t.th[i+1:]...)
			return
		}
	}
}
