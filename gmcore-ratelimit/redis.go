package gmcoreratelimit

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type RedisLimiter struct {
	client *redisGoClient
	rules  map[string]Rule
}

func NewRedisLimiter(cfg RedisConfig, rules map[string]Rule) (*RedisLimiter, error) {
	client, err := newRedisClient(cfg)
	if err != nil {
		return nil, err
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
	return &RedisLimiter{client: client, rules: normalizedRules}, nil
}

func (l *RedisLimiter) Allow(ctx context.Context, ruleName, key string) (bool, error) {
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
	cacheKey := "ratelimit:" + rule.Name + ":" + key

	allowed, _, err := l.client.Allow(ctx, cacheKey, int(rule.Limit), rule.Window)
	return allowed, err
}

func (l *RedisLimiter) Reset(ctx context.Context, ruleName, key string) error {
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
	cacheKey := "ratelimit:" + ruleName + ":" + key
	return l.client.Del(ctx, cacheKey).Err()
}

func (l *RedisLimiter) Close() error {
	if l.client == nil {
		return nil
	}
	return l.client.Close()
}

type redisClient interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error)
	Del(ctx context.Context, keys ...string) redisResult
	Close() error
}

type redisResult interface {
	Err() error
}

type redisGoClient struct {
	client *redis.Client
}

func newRedisClient(cfg RedisConfig) (*redisGoClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &redisGoClient{client: client}, nil
}

func (c *redisGoClient) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, error) {
	now := time.Now().UnixMilli()
	windowMs := window.Milliseconds()
	windowStart := now - windowMs

	pipe := c.client.Pipeline()

	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10))
	countCmd := pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})

	pipe.Expire(ctx, key, window)

	_, err := pipe.Exec(ctx)
	if err != nil && !errors.Is(err, redis.Nil) {
		return true, 0, err
	}

	count := int(countCmd.Val())
	return count < limit, limit - count - 1, nil
}

func (c *redisGoClient) Del(ctx context.Context, keys ...string) redisResult {
	return c.client.Del(ctx, keys...)
}

func (c *redisGoClient) Close() error {
	return c.client.Close()
}
