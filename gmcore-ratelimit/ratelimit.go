package gmcoreratelimit

import (
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	Name      string        `yaml:"name"`
	Limit     int           `yaml:"limit"`
	Window    time.Duration `yaml:"-"`
	RawWindow string        `yaml:"window"`
}

type Config struct {
	Rules map[string]Rule `yaml:"rules"`
}

type configFile struct {
	RateLimit Config `yaml:"rate_limit"`
}

type entry struct {
	Count     int
	ExpiresAt time.Time
}

type Limiter struct {
	mu    sync.Mutex
	rules map[string]Rule
	hits  map[string]entry
	now   func() time.Time
}

func New(cfg Config) *Limiter {
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
	return &Limiter{rules: rules, hits: map[string]entry{}, now: time.Now}
}

func DefaultConfig() Config {
	return Config{Rules: map[string]Rule{
		"security.login":          {Name: "security.login", Limit: 8, Window: time.Minute},
		"security.token":          {Name: "security.token", Limit: 20, Window: time.Minute},
		"security.2fa_challenge":  {Name: "security.2fa_challenge", Limit: 6, Window: time.Minute},
		"security.recovery_codes": {Name: "security.recovery_codes", Limit: 3, Window: 5 * time.Minute},
	}}
}

func Load(paths ...string) (Config, error) {
	cfg := DefaultConfig()
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return cfg, err
		}
		var parsed configFile
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			return cfg, err
		}
		cfg = Merge(cfg, parsed.RateLimit)
	}
	return cfg, nil
}

func Merge(base Config, overlay Config) Config {
	out := Config{Rules: map[string]Rule{}}
	for name, rule := range base.Rules {
		out.Rules[name] = rule
	}
	for name, rule := range overlay.Rules {
		name = strings.TrimSpace(name)
		if name == "" {
			name = strings.TrimSpace(rule.Name)
		}
		if name == "" {
			continue
		}
		existing := out.Rules[name]
		if rule.Limit == 0 {
			rule.Limit = existing.Limit
		}
		if strings.TrimSpace(rule.RawWindow) == "" {
			rule.RawWindow = existing.RawWindow
			rule.Window = existing.Window
		}
		rule.Name = name
		out.Rules[name] = rule
	}
	return out
}

func (l *Limiter) Allow(ruleName, key string) bool {
	if l == nil {
		return true
	}
	rule, ok := l.rules[strings.TrimSpace(ruleName)]
	if !ok {
		return true
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
		return true
	}
	if current.Count >= rule.Limit {
		return false
	}
	current.Count++
	l.hits[cacheKey] = current
	return true
}

func (l *Limiter) gcLocked(now time.Time) {
	for key, current := range l.hits {
		if !current.ExpiresAt.IsZero() && now.After(current.ExpiresAt.Add(time.Minute)) {
			delete(l.hits, key)
		}
	}
}

func ClientKey(r *http.Request, discriminator string) string {
	parts := []string{}
	if r != nil {
		host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
		if err != nil || host == "" {
			host = strings.TrimSpace(r.RemoteAddr)
		}
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
