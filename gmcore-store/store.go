package gmcorestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v4/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

type Driver string

const (
	DriverSQLite   Driver = "sqlite"
	DriverMySQL    Driver = "mysql"
	DriverPostgres Driver = "postgres"
)

type Config struct {
	Driver Driver
	DSN    string
}

type HookOperation string

const (
	HookBeforeCreate HookOperation = "before_create"
	HookAfterCreate  HookOperation = "after_create"
	HookBeforeUpdate HookOperation = "before_update"
	HookAfterUpdate  HookOperation = "after_update"
	HookBeforeDelete HookOperation = "before_delete"
	HookAfterDelete  HookOperation = "after_delete"
)

type HookContext struct {
	Operation HookOperation
	Resource  string
	Key       string
	Payload   map[string]interface{}
}

type Hook func(context.Context, *sql.Tx, HookContext) error

type Hooks struct {
	BeforeCreate []Hook
	AfterCreate  []Hook
	BeforeUpdate []Hook
	AfterUpdate  []Hook
	BeforeDelete []Hook
	AfterDelete  []Hook
}

type Store struct {
	db      *sql.DB
	dialect dialect
}

type User struct {
	Email        string
	PasswordHash string
	Roles        []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserStore struct {
	store *Store
	hooks Hooks
}

type dialect interface {
	driverName() string
	placeholder(int) string
}

type sqliteDialect struct{}
type mysqlDialect struct{}
type postgresDialect struct{}

func Open(cfg Config) (*Store, error) {
	d, err := selectDialect(cfg.Driver)
	if err != nil {
		return nil, err
	}

	dsn := cfg.DSN
	if cfg.Driver == DriverSQLite || cfg.Driver == "" {
		dsn = configureSQLiteDSN(dsn)
	}

	db, err := sql.Open(d.driverName(), dsn)
	if err != nil {
		return nil, err
	}
	if cfg.Driver == DriverSQLite || cfg.Driver == "" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(0)
	} else {
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(30 * time.Minute)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, dialect: d}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) UserStore(hooks Hooks) *UserStore {
	return &UserStore{store: s, hooks: hooks}
}

func (s *UserStore) EnsureSchema(ctx context.Context) error {
	query := `
CREATE TABLE IF NOT EXISTS gmcore_users (
  email VARCHAR(255) PRIMARY KEY,
  password_hash TEXT NOT NULL,
  roles TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
)`
	_, err := s.store.db.ExecContext(ctx, query)
	return err
}

func (s *UserStore) List(ctx context.Context) ([]User, error) {
	rows, err := s.store.db.QueryContext(ctx, `SELECT email, password_hash, roles, created_at, updated_at FROM gmcore_users ORDER BY email ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var entry User
		var rolesJSON string
		if err := rows.Scan(&entry.Email, &entry.PasswordHash, &rolesJSON, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(rolesJSON), &entry.Roles); err != nil {
			return nil, err
		}
		users = append(users, entry)
	}
	return users, rows.Err()
}

func (s *UserStore) FindByEmail(ctx context.Context, email string) (User, bool, error) {
	var entry User
	var rolesJSON string
	err := s.store.db.QueryRowContext(ctx, `SELECT email, password_hash, roles, created_at, updated_at FROM gmcore_users WHERE email = `+s.store.dialect.placeholder(1), strings.ToLower(email)).
		Scan(&entry.Email, &entry.PasswordHash, &rolesJSON, &entry.CreatedAt, &entry.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	if err := json.Unmarshal([]byte(rolesJSON), &entry.Roles); err != nil {
		return User{}, false, err
	}
	return entry, true, nil
}

func (s *UserStore) Create(ctx context.Context, user User) error {
	return s.withTx(ctx, HookContext{
		Operation: HookBeforeCreate,
		Resource:  "user",
		Key:       strings.ToLower(user.Email),
		Payload: map[string]interface{}{
			"email": user.Email,
			"roles": user.Roles,
		},
	}, func(tx *sql.Tx) error {
		rolesJSON, err := json.Marshal(user.Roles)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		_, err = tx.ExecContext(ctx,
			`INSERT INTO gmcore_users (email, password_hash, roles, created_at, updated_at) VALUES (`+s.store.dialect.placeholder(1)+`, `+s.store.dialect.placeholder(2)+`, `+s.store.dialect.placeholder(3)+`, `+s.store.dialect.placeholder(4)+`, `+s.store.dialect.placeholder(5)+`)`,
			strings.ToLower(user.Email), user.PasswordHash, string(rolesJSON), now, now,
		)
		return err
	}, HookAfterCreate)
}

func (s *UserStore) UpdatePassword(ctx context.Context, email, passwordHash string) error {
	return s.withTx(ctx, HookContext{
		Operation: HookBeforeUpdate,
		Resource:  "user",
		Key:       strings.ToLower(email),
		Payload:   map[string]interface{}{"email": email},
	}, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE gmcore_users SET password_hash = `+s.store.dialect.placeholder(1)+`, updated_at = `+s.store.dialect.placeholder(2)+` WHERE email = `+s.store.dialect.placeholder(3),
			passwordHash, time.Now().UTC(), strings.ToLower(email),
		)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf("user not found")
		}
		return nil
	}, HookAfterUpdate)
}

func (s *UserStore) Delete(ctx context.Context, email string) error {
	return s.withTx(ctx, HookContext{
		Operation: HookBeforeDelete,
		Resource:  "user",
		Key:       strings.ToLower(email),
		Payload:   map[string]interface{}{"email": email},
	}, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM gmcore_users WHERE email = `+s.store.dialect.placeholder(1), strings.ToLower(email))
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf("user not found")
		}
		return nil
	}, HookAfterDelete)
}

func (s *UserStore) withTx(ctx context.Context, hookCtx HookContext, run func(*sql.Tx) error, after HookOperation) error {
	tx, err := s.store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := s.runHooks(ctx, tx, hookCtx.Operation, hookCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := run(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := s.runHooks(ctx, tx, after, HookContext{
		Operation: after,
		Resource:  hookCtx.Resource,
		Key:       hookCtx.Key,
		Payload:   hookCtx.Payload,
	}); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *UserStore) runHooks(ctx context.Context, tx *sql.Tx, op HookOperation, hookCtx HookContext) error {
	var hooks []Hook
	switch op {
	case HookBeforeCreate:
		hooks = s.hooks.BeforeCreate
	case HookAfterCreate:
		hooks = s.hooks.AfterCreate
	case HookBeforeUpdate:
		hooks = s.hooks.BeforeUpdate
	case HookAfterUpdate:
		hooks = s.hooks.AfterUpdate
	case HookBeforeDelete:
		hooks = s.hooks.BeforeDelete
	case HookAfterDelete:
		hooks = s.hooks.AfterDelete
	}
	for _, hook := range hooks {
		if err := hook(ctx, tx, hookCtx); err != nil {
			return err
		}
	}
	return nil
}

func selectDialect(driver Driver) (dialect, error) {
	switch driver {
	case DriverSQLite, "":
		return sqliteDialect{}, nil
	case DriverMySQL:
		return mysqlDialect{}, nil
	case DriverPostgres:
		return postgresDialect{}, nil
	default:
		return nil, fmt.Errorf("unsupported driver %q", driver)
	}
}

func (sqliteDialect) driverName() string   { return "sqlite3" }
func (mysqlDialect) driverName() string    { return "mysql" }
func (postgresDialect) driverName() string { return "pgx" }

func (sqliteDialect) placeholder(_ int) string { return "?" }
func (mysqlDialect) placeholder(_ int) string  { return "?" }
func (postgresDialect) placeholder(i int) string {
	return fmt.Sprintf("$%d", i)
}

func configureSQLiteDSN(path string) string {
	if strings.Contains(path, "?") {
		return path
	}
	values := url.Values{}
	values.Set("_busy_timeout", "5000")
	values.Set("_journal_mode", "WAL")
	values.Set("_synchronous", "NORMAL")
	values.Set("_foreign_keys", "on")
	return path + "?" + values.Encode()
}
