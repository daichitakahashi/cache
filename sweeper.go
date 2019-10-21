package cache

import (
	"errors"
	"time"
)

const (
	// DefaultInterval is
	DefaultInterval = time.Second * 30
	// DefaultExpire is
	DefaultExpire = time.Minute * 5
)

// Sweeper is expiration manager
type Sweeper struct {
	Interval time.Duration
	Expire   time.Duration
	Punctual bool
	stop     chan struct{} // stopチャネルは構造体メンバにする必要がない。ので、Sweeper使い回しできるね
}

func (s *Sweeper) checkExpiration(t *Throttle) {
	if s.Interval == 0 {
		s.Interval = DefaultInterval
	}
	if s.Expire == 0 {
		s.Expire = DefaultExpire
	}

	s.stop = make(chan struct{})
	timer := time.NewTicker(s.Interval)
	for {
		select {
		case <-timer.C:
			t.mutex.Lock()
			now := time.Now()
			for key, c := range t.m {
				if ec := c.(*expireCache); ec.isExpired(now) {
					t.terminate(key)
				}
			}
			t.mutex.Unlock()
		case <-s.stop:
			timer.Stop()
			return
		}
	}
}

func (s *Sweeper) touch(c Cache) *expireCache {
	return &expireCache{
		Cache:   c,
		sweeper: s,
		// nextDeadline: time.Now().Add(s.Expire), // touchは必ずロック時に読み込まれなければならないこととする
	}
}

type expireCache struct {
	Cache
	sweeper      *Sweeper
	nextDeadline time.Time
	expired      bool
}

func (e *expireCache) Get() interface{} {
	if e.expired {
		return nil
	}
	value := e.Cache.Get()
	if !e.sweeper.Punctual {
		e.nextDeadline = time.Now().Add(e.sweeper.Expire) // postpone expiration
	}
	return value
}

func (e *expireCache) Reload() error {
	if e.expired {
		return errors.New("Reload: Cache already expired")
	}
	return e.Cache.Reload()
}

func (e *expireCache) Updated() (bool, error) {
	if e.expired {
		return false, errors.New("Updated: Cache already expired")
	}
	return e.Cache.Updated()
}

func (e *expireCache) Release() {
	if e.expired {
		return
	}
	e.Cache.Release()
	e.expired = true
}

func (e *expireCache) isExpired(now time.Time) bool {
	return now.After(e.nextDeadline)
}

var _ Cache = (*expireCache)(nil)
