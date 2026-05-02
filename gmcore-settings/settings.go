package gmcoresettings

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"

	gmerr "github.com/gmcorenet/gmcore-error"
)

type Setting struct {
	Key         string
	Value       string
	Type        string
	Description string
	Editable    bool
	Encrypted   bool
}

type Encryptor interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}

type Config struct {
	DSN       string
	Encryptor Encryptor
}

type Store struct {
	db    *sql.DB
	mu    sync.RWMutex
	cache map[string]Setting
	enc   Encryptor
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	return OpenWithConfig(ctx, Config{DSN: dsn})
}

func OpenWithConfig(ctx context.Context, cfg Config) (*Store, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return nil, errors.New("missing settings dsn")
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.ExecContext(ctx, `PRAGMA journal_mode=WAL;`); err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `PRAGMA busy_timeout=5000;`); err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `PRAGMA synchronous=NORMAL;`); err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS gmcore_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL DEFAULT 'string',
  description TEXT NOT NULL DEFAULT '',
  editable INTEGER NOT NULL DEFAULT 1,
  encrypted INTEGER NOT NULL DEFAULT 0
)`); err != nil {
		return nil, err
	}
	_, _ = db.ExecContext(ctx, `ALTER TABLE gmcore_settings ADD COLUMN description TEXT NOT NULL DEFAULT ''`)
	_, _ = db.ExecContext(ctx, `ALTER TABLE gmcore_settings ADD COLUMN encrypted INTEGER NOT NULL DEFAULT 0`)
	store := &Store{db: db, cache: map[string]Setting{}, enc: cfg.Encryptor}
	if err := store.Reload(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Reload(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value, type, description, editable, encrypted FROM gmcore_settings`)
	if err != nil {
		return err
	}
	defer rows.Close()
	cache := make(map[string]Setting)
	for rows.Next() {
		var current Setting
		var editable int
		var encrypted int
		if err := rows.Scan(&current.Key, &current.Value, &current.Type, &current.Description, &editable, &encrypted); err != nil {
			return err
		}
		current.Editable = editable != 0
		current.Encrypted = encrypted != 0
		if current.Encrypted && s.enc != nil && strings.TrimSpace(current.Value) != "" {
			if plain, err := s.enc.Decrypt(current.Value); err == nil {
				current.Value = plain
			}
		}
		cache[current.Key] = current
	}
	s.mu.Lock()
	s.cache = cache
	s.mu.Unlock()
	return rows.Err()
}

func (s *Store) Get(key string) (Setting, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	current, ok := s.cache[strings.TrimSpace(key)]
	return current, ok
}

func (s *Store) List() []Setting {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Setting, 0, len(s.cache))
	for _, current := range s.cache {
		out = append(out, current)
	}
	return out
}

func (s *Store) ListEditable() []Setting {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Setting, 0, len(s.cache))
	for _, current := range s.cache {
		if current.Editable {
			out = append(out, current)
		}
	}
	return out
}

func (s *Store) GetString(key, fallback string) string {
	if current, ok := s.Get(key); ok && strings.TrimSpace(current.Value) != "" {
		return current.Value
	}
	return fallback
}

func (s *Store) GetBool(key string, fallback bool) bool {
	if current, ok := s.Get(key); ok {
		value := strings.ToLower(strings.TrimSpace(current.Value))
		return value == "1" || value == "true" || value == "yes" || value == "on"
	}
	return fallback
}

func (s *Store) Set(ctx context.Context, key, value, valueType string, editable bool) error {
	return s.SetWithOptions(ctx, key, value, valueType, "", editable, false)
}

func (s *Store) SetDescription(ctx context.Context, key, value, valueType, description string, editable bool) error {
	return s.SetWithOptions(ctx, key, value, valueType, description, editable, false)
}

func (s *Store) SetWithOptions(ctx context.Context, key, value, valueType, description string, editable bool, encrypted bool) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("missing setting key")
	}
	valueType = strings.TrimSpace(valueType)
	if valueType == "" {
		valueType = "string"
	}
	editableInt := 0
	if editable {
		editableInt = 1
	}
	encryptedInt := 0
	storedValue := value
	if encrypted {
		encryptedInt = 1
		if s.enc == nil {
			return errors.New("missing encryptor")
		}
		encoded, err := s.enc.Encrypt(value)
		if err != nil {
			return err
		}
		storedValue = encoded
	}
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO gmcore_settings (key, value, type, description, editable, encrypted)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value, type=excluded.type, description=excluded.description, editable=excluded.editable, encrypted=excluded.encrypted
`, key, storedValue, valueType, strings.TrimSpace(description), editableInt, encryptedInt); err != nil {
		return err
	}
	s.mu.Lock()
	s.cache[key] = Setting{Key: key, Value: value, Type: valueType, Description: strings.TrimSpace(description), Editable: editable, Encrypted: encrypted}
	s.mu.Unlock()
	return nil
}

func (s *Store) Seed(ctx context.Context, key, value, valueType string, editable bool) error {
	return s.SeedWithOptions(ctx, key, value, valueType, "", editable, false)
}

func (s *Store) SeedDescription(ctx context.Context, key, value, valueType, description string, editable bool) error {
	return s.SeedWithOptions(ctx, key, value, valueType, description, editable, false)
}

func (s *Store) SeedWithOptions(ctx context.Context, key, value, valueType, description string, editable bool, encrypted bool) error {
	if current, ok := s.Get(key); ok {
		if current.Type == strings.TrimSpace(valueType) &&
			current.Description == strings.TrimSpace(description) &&
			current.Editable == editable &&
			current.Encrypted == encrypted {
			return nil
		}
		return s.SetWithOptions(ctx, key, current.Value, valueType, description, editable, encrypted)
	}
	return s.SetWithOptions(ctx, key, value, valueType, description, editable, encrypted)
}
