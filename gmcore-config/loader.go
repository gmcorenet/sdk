package gmcore_config

import (
	"os"
	"path/filepath"
)

type Loader[T any] struct {
	appPath string
	env     map[string]string
}

func NewLoader[T any](appPath string) *Loader[T] {
	return &Loader[T]{
		appPath: appPath,
		env:     LoadAppEnv(appPath),
	}
}

func (l *Loader[T]) Load(path string) (*T, error) {
	cfg := new(T)
	opts := Options{
		Env:        l.env,
		Parameters: map[string]string{},
		Strict:     false,
	}
	if err := LoadYAML(path, cfg, opts); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (l *Loader[T]) LoadDefault(fileName string) (*T, error) {
	candidates := []string{
		filepath.Join(l.appPath, "config", fileName),
		filepath.Join(l.appPath, fileName),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return l.Load(path)
		}
	}
	return nil, nil
}
