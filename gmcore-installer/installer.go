package gmcore_installer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gmcorenet/sdk/gmcore-config"
)

type Installer struct {
	rootPath string
	registry *gmcore_config.RecipeRegistry
}

func New(rootPath string) *Installer {
	return &Installer{
		rootPath: rootPath,
		registry: gmcore_config.NewRecipeRegistry(),
	}
}

func (i *Installer) InstallSDKConfigs(vars map[string]string) error {
	recipes := i.registry.GetEnabledRecipes()

	for _, recipe := range recipes {
		files, err := i.registry.RenderRecipe(recipe, vars)
		if err != nil {
			return fmt.Errorf("failed to render recipe %s: %w", recipe.Name, err)
		}

		for _, file := range files {
			fullPath := filepath.Join(i.rootPath, file.Path)

			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", file.Path, err)
			}

			perms := file.Mode
			if perms == 0 {
				perms = 0644
			}

			if err := os.WriteFile(fullPath, []byte(file.Content), perms); err != nil {
				return fmt.Errorf("failed to write %s: %w", file.Path, err)
			}
		}
	}

	return nil
}

func (i *Installer) Install() error {
	dirs := []string{
		"config/packages",
		"public",
		"var/log",
		"var/data",
		"var/cache",
		"var/tmp",
		"var/socket",
		"var/keys",
		"internal/controller",
		"internal/model",
		"internal/service",
		"internal/repository",
		"internal/middleware",
		"translations",
	}

	for _, dir := range dirs {
		path := filepath.Join(i.rootPath, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}

	files := map[string]string{
		"config/app.yaml":           appConfig,
		"config/packages/framework.yaml": frameworkConfig,
		"config/services.yaml":     servicesConfig,
		"public/index.php":         indexPHP,
		"app.yaml":                 appYaml,
	}

	for file, content := range files {
		path := filepath.Join(i.rootPath, file)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", file, err)
		}
	}

	return nil
}

func (i *Installer) CreateBundle(name string) error {
	dir := filepath.Join(i.rootPath, "bundles", name)
	return os.MkdirAll(dir, 0755)
}

func (i *Installer) CreateModule(name string) error {
	dir := filepath.Join(i.rootPath, "internal", name)
	return os.MkdirAll(dir, 0755)
}

func (i *Installer) GetRecipeRegistry() *gmcore_config.RecipeRegistry {
	return i.registry
}

func (i *Installer) EnableSDK(name string) {
	i.registry.Enable(name)
}

func (i *Installer) DisableSDK(name string) {
	i.registry.Disable(name)
}

const appConfig = `app:
  name: "My GMCore Application"
  env: dev
  debug: true
  timezone: UTC

server:
  host: 0.0.0.0
  port: 8080
`

const frameworkConfig = `framework:
  secret: %env(APP_SECRET)%
  session:
    lifetime: 3600
  router:
    resource: config/routes.yaml
`

const servicesConfig = `services:
  _defaults:
    autowire: true
    public: false
`

const indexPHP = `<?php
// GMCore Web Entry Point
echo "GMCore Framework";
`

const appYaml = `name: My GMCore Application
version: 1.0.0
`
