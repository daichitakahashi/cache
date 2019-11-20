package cache

import (
	"fmt"
	"sync"
)

// Throttle is
type Throttle struct {
	factory Factory
	m       map[string]Cache
	th      []string
	mutex   sync.Mutex
	sweeper Sweeper
}

// NewThrottle is
func NewThrottle(factory Factory, capacity int, sweeper Sweeper) *Throttle {
	if sweeper == nil {
		sweeper = nop
	}
	th := &Throttle{
		factory: factory,
		m:       make(map[string]Cache),
		th:      make([]string, 0, capacity),
		sweeper: sweeper,
	}
	sweeper.register(th)
	return th
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
		err = newCache.Reload()
		if err != nil {
			return nil, err
		}
		newCache = t.sweeper.touch(newCache)
		t.m[key] = newCache
		t.shift(key)
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

// Replace is
func (t *Throttle) Replace(key string, v interface{}) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	c, ok := t.m[key]
	if !ok {
		return &OpError{"replace", fmt.Errorf("key %s not found", key)}
	}
	err := c.Replace(v)
	if err != nil {
		return err
	}
	t.shift(key)
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

func (t *Throttle) terminateAll() {
	for _, c := range t.m {
		c.Release()
	}
	t.th = nil
	t.m = nil
}
