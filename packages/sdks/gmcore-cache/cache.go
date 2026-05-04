package gmcore_cache

import (
	"sync"
	"time"
)

type Item interface {
	GetKey() string
	Get() interface{}
	IsHit() bool
	Set(value interface{}) Item
	ExpiresAt(t time.Time) Item
	ExpiresAfter(d time.Duration) Item
	IsExpired() bool
}

type item struct {
	key        string
	value      interface{}
	expiration time.Time
	hit        bool
}

func newItem(key string, value interface{}) *item {
	return &item{key: key, value: value}
}

func (i *item) GetKey() string              { return i.key }
func (i *item) Get() interface{}            { return i.value }
func (i *item) IsHit() bool                 { return i.hit }
func (i *item) Set(v interface{}) Item       { i.value = v; return i }
func (i *item) ExpiresAt(t time.Time) Item  { i.expiration = t; return i }
func (i *item) ExpiresAfter(d time.Duration) Item { i.expiration = time.Now().Add(d); return i }
func (i *item) IsExpired() bool             { return !i.expiration.IsZero() && time.Now().After(i.expiration) }

type Pool interface {
	GetItem(key string) Item
	HasItem(key string) bool
	Clear() bool
	DeleteItem(key string) bool
	Save(Item) bool
}

type ArrayPool struct {
	items map[string]*item
	mu    sync.RWMutex
}

func NewArrayPool() *ArrayPool {
	return &ArrayPool{items: make(map[string]*item)}
}

func (p *ArrayPool) GetItem(key string) Item {
	p.mu.Lock()
	defer p.mu.Unlock()

	if it, ok := p.items[key]; ok && !it.IsExpired() {
		it.hit = true
		return it
	}
	ni := newItem(key, nil)
	ni.hit = false
	return ni
}

func (p *ArrayPool) HasItem(key string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if it, ok := p.items[key]; ok {
		return !it.IsExpired()
	}
	return false
}

func (p *ArrayPool) Clear() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items = make(map[string]*item)
	return true
}

func (p *ArrayPool) DeleteItem(key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.items, key)
	return true
}

func (p *ArrayPool) Save(i Item) bool {
	if i == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if it, ok := i.(*item); ok {
		p.items[i.GetKey()] = it
		return true
	}
	return false
}

type ChainPool struct {
	pools []Pool
	mu    sync.RWMutex
}

func NewChainPool(pools ...Pool) *ChainPool {
	return &ChainPool{pools: pools}
}

func (c *ChainPool) AddPool(pool Pool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pools = append(c.pools, pool)
}

func (c *ChainPool) RemovePool(pool Pool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, p := range c.pools {
		if p == pool {
			c.pools = append(c.pools[:i], c.pools[i+1:]...)
			return
		}
	}
}

func (c *ChainPool) GetItem(key string) Item {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, pool := range c.pools {
		if pool.HasItem(key) {
			return pool.GetItem(key)
		}
	}
	return newItem(key, nil)
}

func (c *ChainPool) HasItem(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, pool := range c.pools {
		if pool.HasItem(key) {
			return true
		}
	}
	return false
}

func (c *ChainPool) Clear() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, pool := range c.pools {
		pool.Clear()
	}
	return true
}

func (c *ChainPool) DeleteItem(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, pool := range c.pools {
		pool.DeleteItem(key)
	}
	return true
}

func (c *ChainPool) Save(i Item) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, pool := range c.pools {
		pool.Save(i)
	}
	return true
}
