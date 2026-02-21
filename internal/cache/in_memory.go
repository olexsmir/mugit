package cache

import (
	"sync"
	"time"
)

type item[T any] struct {
	v      T
	expiry time.Time
}

func (i item[T]) isExpired() bool {
	return time.Now().After(i.expiry)
}

type InMemory[T any] struct {
	mu   sync.RWMutex
	ttl  time.Duration
	data map[string]item[T]
}

func NewInMemory[T any](ttl time.Duration) *InMemory[T] {
	c := &InMemory[T]{
		data: make(map[string]item[T]),
		ttl:  ttl,
	}

	go c.clean()
	return c
}

func (m *InMemory[T]) Set(key string, val T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = item[T]{
		v:      val,
		expiry: time.Now().Add(m.ttl),
	}
}

func (m *InMemory[T]) Get(key string) (T, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	val, found := m.data[key]
	if !found || val.isExpired() {
		var t T
		return t, false
	}
	return val.v, true
}

func (m *InMemory[T]) clean() {
	for range time.Tick(5 * time.Second) {
		m.mu.Lock()
		for k, v := range m.data {
			if v.isExpired() {
				delete(m.data, k)
			}
		}
		m.mu.Unlock()
	}
}
