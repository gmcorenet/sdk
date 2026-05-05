package gmcore_cache

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gmcorenet/sdk/gmcore-config"
)

type Config struct {
	Adapter string                 `yaml:"adapter" json:"adapter"`
	TTL     int                    `yaml:"ttl" json:"ttl"`
	Prefix  string                 `yaml:"prefix" json:"prefix"`
	Params  map[string]interface{} `yaml:"params" json:"params"`
}

type ConfigLoader struct {
	appPath string
	env     map[string]string
}

func NewConfigLoader(appPath string) *ConfigLoader {
	return &ConfigLoader{
		appPath: appPath,
		env:     gmcore_config.LoadAppEnv(appPath),
	}
}

func (l *ConfigLoader) Load(path string) (*Config, error) {
	cfg := &Config{}

	opts := gmcore_config.Options{
		Env:        l.env,
		Parameters: map[string]string{},
		Strict:     false,
	}

	if err := gmcore_config.LoadYAML(path, cfg, opts); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (l *ConfigLoader) LoadDefault() (*Config, error) {
	candidates := []string{
		filepath.Join(l.appPath, "config", "cache.yaml"),
		filepath.Join(l.appPath, "config", "cache.yml"),
		filepath.Join(l.appPath, "cache.yaml"),
		filepath.Join(l.appPath, "cache.yml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return l.Load(path)
		}
	}

	return nil, nil
}

func LoadConfig(appPath string) (*Config, error) {
	loader := NewConfigLoader(appPath)
	return loader.LoadDefault()
}

type CacheManager interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}) error
	Delete(key string) error
	Clear() error
	Has(key string) bool
}

type Manager struct {
	pool   Pool
	prefix string
	ttl    time.Duration
}

func NewManager(pool Pool, prefix string, ttl int) *Manager {
	return &Manager{
		pool:   pool,
		prefix: prefix,
		ttl:    time.Duration(ttl) * time.Second,
	}
}

func (m *Manager) makeKey(key string) string {
	return m.prefix + key
}

func (m *Manager) Get(key string) (interface{}, bool) {
	item := m.pool.GetItem(m.makeKey(key))
	if item.IsHit() {
		return item.Get(), true
	}
	return nil, false
}

func (m *Manager) Set(key string, value interface{}) error {
	i := newItem(m.makeKey(key), value)
	if m.ttl > 0 {
		i.ExpiresAfter(m.ttl)
	}
	m.pool.Save(i)
	return nil
}

func (m *Manager) Delete(key string) error {
	m.pool.DeleteItem(m.makeKey(key))
	return nil
}

func (m *Manager) Clear() error {
	m.pool.Clear()
	return nil
}

func (m *Manager) Has(key string) bool {
	return m.pool.HasItem(m.makeKey(key))
}

type MemoryAdapter struct {
	pool *ArrayPool
}

func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{
		pool: NewArrayPool(),
	}
}

func (a *MemoryAdapter) Get(key string) (interface{}, bool) {
	item := a.pool.GetItem(key)
	return item.Get(), item.IsHit()
}

func (a *MemoryAdapter) Set(key string, value interface{}) error {
	i := newItem(key, value)
	a.pool.Save(i)
	return nil
}

func (a *MemoryAdapter) Delete(key string) error {
	a.pool.DeleteItem(key)
	return nil
}

func (a *MemoryAdapter) Clear() error {
	a.pool.Clear()
	return nil
}

func (a *MemoryAdapter) Has(key string) bool {
	return a.pool.HasItem(key)
}

type ManagerFactory func(cfg *Config) (CacheManager, error)

var adapters = map[string]ManagerFactory{
	"memory": func(cfg *Config) (CacheManager, error) {
		pool := NewArrayPool()
		return &Manager{
			pool:   pool,
			prefix: getString(cfg.Prefix, "cache_"),
			ttl:    time.Duration(getInt(cfg.TTL, 3600)) * time.Second,
		}, nil
	},
}

func RegisterAdapter(name string, factory ManagerFactory) {
	adapters[name] = factory
}

func CreateManager(cfg *Config) (CacheManager, error) {
	adapter := getString(cfg.Adapter, "memory")
	factory, ok := adapters[adapter]
	if !ok {
		return nil, ErrAdapterNotFound
	}
	return factory(cfg)
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

type Item interface {
	GetKey() string
	Get() interface{}
	IsHit() bool
	Set(value interface{}) Item
	ExpiresAt(t time.Time) Item
	ExpiresAfter(d time.Duration) Item
	IsExpired() bool
}

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

var ErrAdapterNotFound = &AdapterError{Message: "cache adapter not found"}

type AdapterError struct {
	Message string
}

func (e *AdapterError) Error() string {
	return e.Message
}

func getString(val interface{}, defaultVal string) string {
	if s, ok := val.(string); ok {
		return s
	}
	return defaultVal
}

func getInt(val interface{}, defaultVal int) int {
	if i, ok := val.(int); ok {
		return i
	}
	return defaultVal
}
