package gmcoreratelimit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type PostgresConfig struct {
	DSN        string
	PoolSize   int
	MaxRetries int
}

type PostgresLimiter struct {
	db    *sql.DB
	rules map[string]Rule
}

func NewPostgresLimiter(cfg PostgresConfig, rules map[string]Rule) (*PostgresLimiter, error) {
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, err
	}
	if cfg.PoolSize > 0 {
		db.SetMaxOpenConns(cfg.PoolSize)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgres connection failed: %w", err)
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

	limiter := &PostgresLimiter{db: db, rules: normalizedRules}
	if err := limiter.ensureSchema(); err != nil {
		return nil, err
	}

	return limiter, nil
}

func (l *PostgresLimiter) ensureSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS gmcore_ratelimit (
		key TEXT NOT NULL,
		rule_name TEXT NOT NULL,
		counter INTEGER NOT NULL DEFAULT 0,
		window_start TIMESTAMP NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP NOT NULL,
		PRIMARY KEY (key, rule_name)
	);
	CREATE INDEX IF NOT EXISTS idx_ratelimit_expires ON gmcore_ratelimit(expires_at);
	`
	_, err := l.db.Exec(schema)
	return err
}

func (l *PostgresLimiter) Allow(ctx context.Context, ruleName, key string) (bool, error) {
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
	ruleName = rule.Name
	windowStart := time.Now().Add(-rule.Window)
	expiresAt := time.Now().Add(rule.Window)

	var currentCount int
	err := l.db.QueryRowContext(ctx,
		`SELECT counter FROM gmcore_ratelimit WHERE key = $1 AND rule_name = $2 AND window_start > $3`,
		key, ruleName, windowStart,
	).Scan(&currentCount)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}

	if currentCount >= rule.Limit {
		return false, nil
	}

	_, err = l.db.ExecContext(ctx,
		`INSERT INTO gmcore_ratelimit (key, rule_name, counter, window_start, expires_at)
		VALUES ($1, $2, 1, NOW(), $3)
		ON CONFLICT (key, rule_name) DO UPDATE SET
			counter = gmcore_ratelimit.counter + 1,
			window_start = NOW(),
			expires_at = $3`,
		key, ruleName, expiresAt,
	)
	if err != nil {
		return true, nil
	}

	return true, nil
}

func (l *PostgresLimiter) Reset(ctx context.Context, ruleName, key string) error {
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
	_, err := l.db.ExecContext(ctx,
		`DELETE FROM gmcore_ratelimit WHERE key = $1 AND rule_name = $2`,
		key, ruleName,
	)
	return err
}

func (l *PostgresLimiter) Close() error {
	if l.db == nil {
		return nil
	}
	return l.db.Close()
}

type MySQLConfig struct {
	DSN        string
	PoolSize   int
	MaxRetries int
}

type MySQLLimiter struct {
	db    *sql.DB
	rules map[string]Rule
}

func NewMySQLLimiter(cfg MySQLConfig, rules map[string]Rule) (*MySQLLimiter, error) {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, err
	}
	if cfg.PoolSize > 0 {
		db.SetMaxOpenConns(cfg.PoolSize)
		db.SetMaxIdleConns(cfg.PoolSize)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("mysql connection failed: %w", err)
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

	limiter := &MySQLLimiter{db: db, rules: normalizedRules}
	if err := limiter.ensureSchema(); err != nil {
		return nil, err
	}

	return limiter, nil
}

func (l *MySQLLimiter) ensureSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS gmcore_ratelimit (
		\`key\` VARCHAR(255) NOT NULL,
		rule_name VARCHAR(255) NOT NULL,
		counter INT NOT NULL DEFAULT 0,
		window_start TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		expires_at TIMESTAMP NOT NULL,
		PRIMARY KEY (\`key\`, rule_name),
		INDEX idx_expires (expires_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`
	_, err := l.db.Exec(schema)
	return err
}

func (l *MySQLLimiter) Allow(ctx context.Context, ruleName, key string) (bool, error) {
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
	ruleName = rule.Name
	windowStart := time.Now().Add(-rule.Window)
	expiresAt := time.Now().Add(rule.Window)

	var currentCount int
	err := l.db.QueryRowContext(ctx,
		`SELECT counter FROM gmcore_ratelimit WHERE \`key\` = ? AND rule_name = ? AND window_start > ?`,
		key, ruleName, windowStart,
	).Scan(&currentCount)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}

	if currentCount >= rule.Limit {
		return false, nil
	}

	_, err = l.db.ExecContext(ctx,
		`INSERT INTO gmcore_ratelimit (\`key\`, rule_name, counter, window_start, expires_at)
		VALUES (?, ?, 1, NOW(), ?)
		ON DUPLICATE KEY UPDATE
			counter = gmcore_ratelimit.counter + 1,
			window_start = NOW(),
			expires_at = ?`,
		key, ruleName, expiresAt, expiresAt,
	)
	if err != nil {
		return true, nil
	}

	return true, nil
}

func (l *MySQLLimiter) Reset(ctx context.Context, ruleName, key string) error {
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
	_, err := l.db.ExecContext(ctx,
		`DELETE FROM gmcore_ratelimit WHERE \`key\` = ? AND rule_name = ?`,
		key, ruleName,
	)
	return err
}

func (l *MySQLLimiter) Close() error {
	if l.db == nil {
		return nil
	}
	return l.db.Close()
}
