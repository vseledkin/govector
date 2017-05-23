package govector

import (
	"sync"
	"time"
)

type Cache struct {
	mu    sync.RWMutex
	cache map[string][]float32
}

func NewCache() *Cache {
	cache := new(Cache)
	cache.cache = make(map[string][]float32)
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for range ticker.C {
			cache.mu.Lock()
			//fmt.Printf("Cache: %d\n", len(cache.cache))
			cache.cache = make(map[string][]float32)
			cache.mu.Unlock()
		}
	}()

	return cache
}

func (c *Cache) Get(k string) (v []float32, ok bool) {
	c.mu.Lock()
	v, ok = c.cache[k]
	c.mu.Unlock()
	return
}

func (c *Cache) Set(k string, v []float32) {
	c.mu.Lock()
	c.cache[k] = v
	c.mu.Unlock()
	return
}
