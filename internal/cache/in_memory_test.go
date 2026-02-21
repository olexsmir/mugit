package cache

import (
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"olexsmir.xyz/x/is"
)

func TestInMemory_Set(t *testing.T) {
	c := NewInMemory[string](time.Minute)
	t.Run("sets", func(t *testing.T) {
		c.Set("asdf", "qwer")
		is.Equal(t, c.data["asdf"].v, "qwer")
	})
	t.Run("overwrites prev value", func(t *testing.T) {
		c.Set("asdf", "one")
		c.Set("asdf", "two")
		is.Equal(t, c.data["asdf"].v, "two")
	})
}

func TestInMemory_Get(t *testing.T) {
	c := NewInMemory[string](time.Minute)

	t.Run("hit", func(t *testing.T) {
		c.Set("asdf", "qwer")
		v, found := c.Get("asdf")
		is.Equal(t, true, found)
		is.Equal(t, "qwer", v)
	})
	t.Run("miss", func(t *testing.T) {
		_, found := c.Get("missing")
		is.Equal(t, false, found)
	})
	t.Run("expired item", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			c.Set("asdf", "qwer")
			time.Sleep(2 * time.Minute)
			v, found := c.Get("asdf")
			is.Equal(t, false, found)
			is.Equal(t, "", v)
		})
	})
}

func TestInMemory_ConcurrentSetGet(t *testing.T) {
	c := NewInMemory[int](time.Minute)
	synctest.Test(t, func(t *testing.T) {
		var wg sync.WaitGroup
		for i := range 50 {
			key := fmt.Sprintf("key-%d", i)
			wg.Go(func() { c.Set(key, i) })
			wg.Go(func() { c.Get(key) })
		}
		wg.Wait()
	})
}
