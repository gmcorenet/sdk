package gmcore_mailer

import (
	"os"
	"path/filepath"

	"github.com/gmcorenet/sdk/gmcore-config"
)

type Config struct {
	Host      string `yaml:"host" json:"host"`
	Port      int    `yaml:"port" json:"port"`
	Username  string `yaml:"username" json:"username"`
	Password  string `yaml:"password" json:"password"`
	From      string `yaml:"from" json:"from"`
	FromName  string `yaml:"from_name" json:"from_name"`
	ReplyTo   string `yaml:"reply_to" json:"reply_to"`
	Encryption string `yaml:"encryption" json:"encryption"`
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
		filepath.Join(l.appPath, "config", "mailer.yaml"),
		filepath.Join(l.appPath, "config", "mailer.yml"),
		filepath.Join(l.appPath, "config", "email.yaml"),
		filepath.Join(l.appPath, "config", "email.yml"),
		filepath.Join(l.appPath, "mailer.yaml"),
		filepath.Join(l.appPath, "mailer.yml"),
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

func (c *Config) Build() *SMTPMailer {
	return NewSMTPMailer(c.Host, c.Port, c.Username, c.Password)
}
