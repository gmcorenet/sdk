package gmcoreratelimit

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

var ErrNotStored = errors.New("key not found")

type Rule struct {
	Name      string
	Limit     int
	Window    time.Duration
	RawWindow string
}

type Config struct {
	Rules map[string]Rule
}

type Limiter interface {
	Allow(ctx context.Context, ruleName, key string) (bool, error)
	Reset(ctx context.Context, ruleName, key string) error
	Close() error
}

type entry struct {
	Count     int
	ExpiresAt time.Time
}

type MemoryLimiter struct {
	mu    sync.Mutex
	rules map[string]Rule
	hits  map[string]entry
	now   func() time.Time
}

func NewMemoryLimiter(cfg Config) *MemoryLimiter {
	rules := map[string]Rule{}
	for name, rule := range cfg.Rules {
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
		rules[normalized] = rule
	}
	return &MemoryLimiter{rules: rules, hits: map[string]entry{}, now: time.Now}
}

func DefaultConfig() Config {
	return Config{Rules: map[string]Rule{
		"security.login":           {Name: "security.login", Limit: 8, Window: time.Minute},
		"security.token":           {Name: "security.token", Limit: 20, Window: time.Minute},
		"security.2fa_challenge":  {Name: "security.2fa_challenge", Limit: 6, Window: time.Minute},
		"security.recovery_codes":  {Name: "security.recovery_codes", Limit: 3, Window: 5 * time.Minute},
	}}
}

func (l *MemoryLimiter) Allow(ctx context.Context, ruleName, key string) (bool, error) {
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
	now := l.now().UTC()
	cacheKey := rule.Name + ":" + key
	l.mu.Lock()
	defer l.mu.Unlock()
	current := l.hits[cacheKey]
	if current.ExpiresAt.IsZero() || !now.Before(current.ExpiresAt) {
		l.hits[cacheKey] = entry{Count: 1, ExpiresAt: now.Add(rule.Window)}
		l.gcLocked(now)
		return true, nil
	}
	if current.Count >= rule.Limit {
		return false, nil
	}
	current.Count++
	l.hits[cacheKey] = current
	return true, nil
}

func (l *MemoryLimiter) Reset(ctx context.Context, ruleName, key string) error {
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
	cacheKey := ruleName + ":" + key
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, cacheKey)
	return nil
}

func (l *MemoryLimiter) Close() error {
	return nil
}

func (l *MemoryLimiter) gcLocked(now time.Time) {
	for key, current := range l.hits {
		if !current.ExpiresAt.IsZero() && now.After(current.ExpiresAt.Add(time.Minute)) {
			delete(l.hits, key)
		}
	}
}

func ClientKey(r *http.Request, discriminator string) string {
	parts := []string{}
	if r != nil {
		host, _, err := strings.Cut(r.RemoteAddr, ":")
		if err != nil {
			host = r.RemoteAddr
		}
		host = strings.TrimSpace(host)
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			host = strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
		if host != "" {
			parts = append(parts, host)
		}
	}
	if value := strings.ToLower(strings.TrimSpace(discriminator)); value != "" {
		parts = append(parts, value)
	}
	return strings.Join(parts, "|")
}
