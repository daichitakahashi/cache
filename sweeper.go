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

// SweepMode :
type SweepMode uint8

// Sweep mode
const (
	SweepPunctual SweepMode = 1 << iota
	SweepConcurrent
)

// Sweeper :
type Sweeper interface {
	Stop()
	touch(c Cache) Cache
	register(t *Throttle)
}

// NewSweeper :
func NewSweeper(interval, expire time.Duration, mode SweepMode) Sweeper {
	if interval <= 0 {
		interval = DefaultInterval
	}
	if expire <= 0 {
		expire = DefaultExpire
	}

	punctual := mode&SweepPunctual == SweepPunctual
	concurrent := mode&SweepConcurrent == SweepConcurrent

	sw := &sweeper{
		throttles:  make([]*Throttle, 0, 5),
		interval:   interval,
		expire:     int64(expire),
		stopCh:     make(chan struct{}),
		punctual:   punctual,
		concurrent: concurrent,
	}
	go sw.start()
	return sw
}

type sweeper struct {
	throttles []*Throttle
	interval  time.Duration
	expire    int64
	stopCh    chan struct{}

	punctual   bool
	concurrent bool
}

func (s *sweeper) start() {
	timer := time.NewTicker(s.interval)
	for {
		select {
		case <-timer.C:
			for _, th := range s.throttles {
				if s.concurrent {
					go s.sweep(th)
				} else {
					s.sweep(th)
				}
			}

		case <-s.stopCh:
			timer.Stop()
			s.stop()
			return
		}
	}
}

func (s *sweeper) sweep(th *Throttle) {
	th.mutex.Lock()
	now := time.Now().UnixNano()
	for key, c := range th.m {
		if c.(*expireCache).isExpired(now) {
			th.terminate(key)
		}
	}
	th.mutex.Unlock()
}

func (s *sweeper) Stop() {
	close(s.stopCh)
}

func (s *sweeper) stop() {
	for _, th := range s.throttles {
		th.terminateAll()
	}
}

func (s *sweeper) register(t *Throttle) {
	s.throttles = append(s.throttles, t)
}

func (s *sweeper) touch(c Cache) Cache {
	return &expireCache{
		Cache: c,
	}
}

type expireCache struct {
	Cache
	sweeper  *sweeper
	deadline int64
}

func (e *expireCache) Get() interface{} {
	if e.deadline < 0 {
		return nil
	}
	value := e.Cache.Get()
	if !e.sweeper.punctual {
		e.deadline += e.sweeper.expire // postpone expiration
	}
	return value
}

func (e *expireCache) Reload() error {
	if e.deadline < 0 {
		return errors.New("cache already expired")
	}
	return e.Cache.Reload()
}

func (e *expireCache) Updated() (bool, error) {
	if e.deadline < 0 {
		return false, errors.New("cache already expired")
	}
	return e.Cache.Updated()
}

func (e *expireCache) Replace(v interface{}) error {
	if e.deadline < 0 {
		return errors.New("cache already expired")
	}
	err := e.Cache.Replace(v)
	if err != nil {
		return err
	}
	if !e.sweeper.punctual {
		e.deadline += e.sweeper.expire // postpone expiration
	}
	return nil
}

func (e *expireCache) Release() {
	if e.deadline < 0 {
		return
	}
	e.Cache.Release()
	e.deadline = -1
	e.Cache = nil
}

func (e *expireCache) isExpired(now int64) (expired bool) {
	if e.deadline <= 0 {
		expired = true
	} else {
		expired = now > e.deadline
	}
	return
}

var _ Cache = (*expireCache)(nil)

type nopSweeper struct{}

func (*nopSweeper) Stop() {}

func (*nopSweeper) register(*Throttle) {}

func (*nopSweeper) touch(c Cache) Cache { return c }

var nop Sweeper = &nopSweeper{}
