package gmcore_transport

import (
	"os"
	"path/filepath"

	"github.com/gmcorenet/sdk/gmcore-config"
)

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

func (l *ConfigLoader) Load(path string) (*FullConfig, error) {
	cfg := &FullConfig{}

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

func (l *ConfigLoader) LoadDefault() (*FullConfig, error) {
	candidates := []string{
		filepath.Join(l.appPath, "config", "transport.yaml"),
		filepath.Join(l.appPath, "config", "transport.yml"),
		filepath.Join(l.appPath, "config", "server.yaml"),
		filepath.Join(l.appPath, "config", "server.yml"),
		filepath.Join(l.appPath, "transport.yaml"),
		filepath.Join(l.appPath, "transport.yml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return l.Load(path)
		}
	}

	return nil, nil
}

func LoadConfig(appPath string) (*FullConfig, error) {
	loader := NewConfigLoader(appPath)
	return loader.LoadDefault()
}

func (c *FullConfig) ToConfig() Config {
	cfg := Config{
		KeysDir: filepath.Join(filepath.Dir(c.Server.UDS.Path), "keys"),
	}

	switch c.Server.Mode {
	case ModeUDS:
		cfg.Mode = ModeUDS
		cfg.Path = c.Server.UDS.Path
	case ModeTCP:
		cfg.Mode = ModeTCP
		cfg.Host = c.Server.TCP.Host
		cfg.Ports = c.Server.TCP.Ports
	case ModeBoth:
		cfg.Mode = ModeBoth
		cfg.Path = c.Server.UDS.Path
		cfg.Host = c.Server.TCP.Host
		cfg.Ports = c.Server.TCP.Ports
	default:
		cfg.Mode = c.Server.Mode
		cfg.Path = c.Server.UDS.Path
		cfg.Host = c.Server.TCP.Host
		cfg.Ports = c.Server.TCP.Ports
	}

	return cfg
}

func (c *FullConfig) ToSecurityProvider() SecurityProvider {
	switch c.Security.Type {
	case "hmac":
		return NewHMACSecurity([]byte(c.Security.Key))
	case "mutual":
		sec, err := NewMutualSecurity(c.Security.CertDir)
		if err != nil {
			return &NoOpSecurity{}
		}
		return sec
	default:
		return &NoOpSecurity{}
	}
}
