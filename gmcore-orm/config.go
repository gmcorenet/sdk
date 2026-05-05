package gmcore_orm

import (
	"os"
	"path/filepath"

	"github.com/gmcorenet/sdk/gmcore-config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Driver     string         `yaml:"driver" json:"driver"`
	DSN       string         `yaml:"dsn" json:"dsn"`
	Pool       PoolConfig     `yaml:"pool" json:"pool"`
	AutoMigrate bool         `yaml:"auto_migrate" json:"auto_migrate"`
	Logging    LoggingConfig  `yaml:"logging" json:"logging"`
}

type PoolConfig struct {
	MaxOpenConns    int `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	ConnMaxIdleTime int `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`
}

type LoggingConfig struct {
	Level      string `yaml:"level" json:"level"`
	SlowThreshold int `yaml:"slow_threshold" json:"slow_threshold"`
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
		filepath.Join(l.appPath, "config", "database.yaml"),
		filepath.Join(l.appPath, "config", "database.yml"),
		filepath.Join(l.appPath, "config", "orm.yaml"),
		filepath.Join(l.appPath, "config", "orm.yml"),
		filepath.Join(l.appPath, "database.yaml"),
		filepath.Join(l.appPath, "database.yml"),
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

func (c *Config) Open() (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch c.Driver {
	case "mysql":
		dialector = mysql.Open(c.DSN)
	case "postgres", "postgresql":
		dialector = postgres.Open(c.DSN)
	case "sqlite":
		dialector = sqlite.Open(c.DSN)
	case "sqlserver":
		dialector = sqlserver.Open(c.DSN)
	default:
		return nil, ErrUnknownDriver
	}

	gormConfig := &gorm.Config{}

	if c.Logging.Level != "" {
		gormConfig.Logger = logger.Default.LogMode(parseLogLevel(c.Logging.Level))
	}

	if c.Logging.SlowThreshold > 0 {
		gormConfig.Logger = logger.Default.LogMode(logger.Slow)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if c.Pool.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(c.Pool.MaxOpenConns)
	}
	if c.Pool.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(c.Pool.MaxIdleConns)
	}
	if c.Pool.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(parseDuration(c.Pool.ConnMaxLifetime))
	}
	if c.Pool.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(parseDuration(c.Pool.ConnMaxIdleTime))
	}

	return db, nil
}

func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Info
	}
}

func parseDuration(seconds int) int {
	return seconds * 1000
}

var ErrUnknownDriver = &ORMError{Message: "unknown database driver"}

type ORMError struct {
	Message string
}

func (e *ORMError) Error() string {
	return e.Message
}
