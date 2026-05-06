package gmcore_i18n

import (
	"errors"

	"github.com/gmcorenet/sdk/gmcore-config"
)

type Config struct {
	DefaultLocale string   `yaml:"default_locale" json:"default_locale"`
	Directories   []string `yaml:"directories" json:"directories"`
	FallbackLocale string `yaml:"fallback_locale" json:"fallback_locale"`
}

func LoadConfig(appPath string) (*Config, error) {
	l := gmcore_config.NewLoader[Config](appPath)
	for _, name := range []string{"i18n.yaml", "i18n.yml", "translation.yaml", "translation.yml"} {
		if cfg, err := l.LoadDefault(name); cfg != nil || err != nil {
			return cfg, err
		}
	}
	return nil, nil
}

func (c *Config) Build() (*Translator, error) {
	if len(c.Directories) == 0 {
		return nil, errors.New("i18n: at least one translation directory is required")
	}

	defaultLocale := c.DefaultLocale
	if defaultLocale == "" {
		defaultLocale = "en"
	}

	return LoadDirs(c.Directories, defaultLocale)
}
