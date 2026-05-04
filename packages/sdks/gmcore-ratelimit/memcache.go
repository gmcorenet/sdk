package gmcore_ratelimit

import (
	"context"
	"strings"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

type MemcacheConfig struct {
	Addr    string
	Timeout time.Duration
}

type MemcacheLimiter struct {
	client *memcache.Client
	rules  map[string]Rule
}

func NewMemcacheLimiter(cfg MemcacheConfig, rules map[string]Rule) (*MemcacheLimiter, error) {
	client := memcache.New(cfg.Addr)
	if cfg.Timeout > 0 {
		client.Timeout = cfg.Timeout
	}
	normalizedRules := map[string]Rule{}
	for name, rule := range rules {
		normalized := strings.TrimSpace(name)
		if normalized == "" {
			normalized = strings.TrimSpace(rule.Name)
		}
		if normalized == "" {
			continue
		}
		if rule.Limit <= 0 {
			rule.Limit = 5
		}
		if rule.Window <= 0 {
			if parsed, err := time.ParseDuration(strings.TrimSpace(rule.RawWindow)); err == nil && parsed > 0 {
				rule.Window = parsed
			}
		}
		if rule.Window <= 0 {
			rule.Window = time.Minute
		}
		rule.Name = normalized
		normalizedRules[normalized] = rule
	}
	return &MemcacheLimiter{client: client, rules: normalizedRules}, nil
}

func (l *MemcacheLimiter) Allow(ctx context.Context, ruleName, key string) (bool, error) {
	if l == nil {
		return true, nil
	}
	rule, ok := l.rules[strings.TrimSpace(ruleName)]
	if !ok {
		return true, nil
	}
	key = strings.TrimSpace(key)
	if key == "" {
		key = "anonymous"
	}
	cacheKey := "rl:" + rule.Name + ":" + key

	current, err := l.client.Get(cacheKey)
	if err != nil && err != memcache.ErrCacheMiss {
		return true, nil
	}

	count := 0
	if current != nil && len(current.Value) > 0 {
		count = int(current.Value[0])
	}

	if count >= rule.Limit {
		return false, nil
	}

	count++
	expiration := int32(rule.Window.Seconds())
	item := &memcache.Item{Key: cacheKey, Value: []byte{byte(count)}, Expiration: expiration}
	l.client.Set(item)

	return true, nil
}

func (l *MemcacheLimiter) Reset(ctx context.Context, ruleName, key string) error {
	if l == nil {
		return nil
	}
	key = strings.TrimSpace(key)
	if key == "" {
		key = "anonymous"
	}
	ruleName = strings.TrimSpace(ruleName)
	if ruleName == "" {
		return nil
	}
	cacheKey := "rl:" + ruleName + ":" + key
	return l.client.Delete(cacheKey)
}

func (l *MemcacheLimiter) Close() error {
	return nil
}
