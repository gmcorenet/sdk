package gmcore_orm

import (
	"time"

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

func LoadConfig(appPath string) (*Config, error) {
	l := gmcore_config.NewLoader[Config](appPath)
	for _, name := range []string{"database.yaml", "database.yml", "orm.yaml", "orm.yml"} {
		if cfg, err := l.LoadDefault(name); cfg != nil || err != nil {
			return cfg, err
		}
	}
	return nil, nil
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
		gormConfig.Logger = logger.Default.LogMode(logger.Warn)
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

func parseDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}

var ErrUnknownDriver = &ORMError{Message: "unknown database driver"}

type ORMError struct {
	Message string
}

func (e *ORMError) Error() string {
	return e.Message
}
