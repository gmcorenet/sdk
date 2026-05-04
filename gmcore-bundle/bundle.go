package gmcore_bundle

import (
	"context"
	"strings"
)

type Bundle interface {
	Name() string
	Boot(ctx context.Context) error
	Shutdown() error
}

type BundleManager struct {
	bundles map[string]Bundle
}

func NewManager() *BundleManager {
	return &BundleManager{bundles: make(map[string]Bundle)}
}

func (m *BundleManager) Register(b Bundle) error {
	m.bundles[b.Name()] = b
	return nil
}

func (m *BundleManager) Get(name string) Bundle {
	return m.bundles[name]
}

func (m *BundleManager) All() map[string]Bundle {
	result := make(map[string]Bundle, len(m.bundles))
	for k, v := range m.bundles {
		result[k] = v
	}
	return result
}

func (m *BundleManager) BootAll(ctx context.Context) error {
	for _, b := range m.bundles {
		if err := b.Boot(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *BundleManager) ShutdownAll() error {
	for _, b := range m.bundles {
		if err := b.Shutdown(); err != nil {
			return err
		}
	}
	return nil
}

type BaseBundle struct{}

func (b *BaseBundle) Boot(ctx context.Context) error { return nil }
func (b *BaseBundle) Shutdown() error               { return nil }

func NewBaseBundle(name string) *BaseBundle {
	return &BaseBundle{}
}

func (b *BaseBundle) Name() string { return "" }

type BootstrapSpec struct {
	Import string `yaml:"import"`
}

type RecipeSpec struct {
	File string `yaml:"file"`
}

type InstallSpec struct {
	Entities  string `yaml:"entities"`
	Examples  string `yaml:"examples"`
	Config    string `yaml:"config"`
	Migrations string `yaml:"migrations"`
}

type Manifest struct {
	Name     string       `yaml:"name"`
	Package  string       `yaml:"package"`
	Module   string       `yaml:"module"`
	Root     string       `yaml:"-"`
	Recipe   RecipeSpec   `yaml:"recipe"`
	Install  InstallSpec  `yaml:"install"`
	Bootstrap BootstrapSpec `yaml:"bootstrap"`
}

func (m Manifest) BootstrapImportPath() string {
	importPath := m.Module
	if importPath == "" {
		importPath = m.Name
	}
	if m.Bootstrap.Import != "" {
		importPath = importPath + "/" + m.Bootstrap.Import
	}
	if strings.HasPrefix(importPath, "gmcore/") {
		importPath = strings.Replace(importPath, "gmcore/", "gmcore-", 1)
	}
	return importPath
}
