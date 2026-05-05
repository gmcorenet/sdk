package gmcore_session

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gmcorenet/sdk/gmcore-config"
)

type Config struct {
	Name      string        `yaml:"name" json:"name"`
	Lifetime  int          `yaml:"lifetime" json:"lifetime"`
	Path      string        `yaml:"path" json:"path"`
	Domain    string        `yaml:"domain" json:"domain"`
	Secure    bool         `yaml:"secure" json:"secure"`
	HTTPOnly  bool         `yaml:"http_only" json:"http_only"`
	SameSite  string        `yaml:"same_site" json:"same_site"`
	Cookie    CookieConfig  `yaml:"cookie" json:"cookie"`
}

type CookieConfig struct {
	Name     string `yaml:"name" json:"name"`
	Path     string `yaml:"path" json:"path"`
	Domain   string `yaml:"domain" json:"domain"`
	Secure   bool   `yaml:"secure" json:"secure"`
	HTTPOnly bool   `yaml:"http_only" json:"http_only"`
	SameSite string `yaml:"same_site" json:"same_site"`
	MaxAge   int    `yaml:"max_age" json:"max_age"`
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
		filepath.Join(l.appPath, "config", "session.yaml"),
		filepath.Join(l.appPath, "config", "session.yml"),
		filepath.Join(l.appPath, "session.yaml"),
		filepath.Join(l.appPath, "session.yml"),
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

func (c *Config) ApplyTo(manager *Manager) error {
	if c.Name != "" && c.Name != manager.Name() {
		return errors.New("session manager name cannot be changed after creation, recreate manager")
	}
	return nil
}

func (c *Config) ToCookieConfig() CookieConfig {
	if c.Cookie.Name != "" {
		return c.Cookie
	}
	return CookieConfig{
		Name:     c.Name,
		Path:     c.Path,
		Domain:   c.Domain,
		Secure:   c.Secure,
		HTTPOnly: c.HTTPOnly,
		SameSite: c.SameSite,
		MaxAge:   c.Lifetime,
	}
}

func SameSiteFromString(s string) http.SameSite {
	switch s {
	case "strict", "Strict":
		return http.SameSiteStrictMode
	case "lax", "Lax":
		return http.SameSiteLaxMode
	case "none", "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}

type ManagerConfig struct {
	Store    Store
	Name     string
	Lifetime time.Duration
	Cookie   CookieConfig
}

func NewManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		Name:     "gmcore_session",
		Lifetime: 3600 * time.Second,
		Cookie: CookieConfig{
			Path:     "/",
			Secure:   true,
			HTTPOnly: true,
			SameSite: "strict",
			MaxAge:   3600,
		},
	}
}

func (c *ManagerConfig) WithStore(store Store) *ManagerConfig {
	c.Store = store
	return c
}

func (c *ManagerConfig) WithName(name string) *ManagerConfig {
	c.Name = name
	c.Cookie.Name = name
	return c
}

func (c *ManagerConfig) WithLifetime(seconds int) *ManagerConfig {
	c.Lifetime = time.Duration(seconds) * time.Second
	c.Cookie.MaxAge = seconds
	return c
}

func (c *ManagerConfig) Build() *Manager {
	return NewManager(c.Store, c.Name, c.Lifetime)
}
