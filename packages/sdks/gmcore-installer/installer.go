package gmcore_installer

import (
	"fmt"
	"os"
	"path/filepath"
)

type Installer struct {
	rootPath string
}

func New(rootPath string) *Installer {
	return &Installer{rootPath: rootPath}
}

func (i *Installer) Install() error {
	dirs := []string{
		"config/packages",
		"public",
		"var/log",
		"var/data",
		"var/cache",
		"var/tmp",
		"internal/controller",
		"internal/model",
		"internal/service",
		"internal/repository",
		"internal/middleware",
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
		"config/routes.yaml":        routesConfig,
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

const appConfig = `# GMCore Application Configuration
app:
  name: "GMCore Application"
  env: dev
  debug: true
  timezone: UTC

server:
  host: 0.0.0.0
  port: 8080
`

const frameworkConfig = `framework:
  secret: "%env(APP_SECRET)%"
  session:
    lifetime: 3600
  router:
    resource: config/routes.yaml
`

const routesConfig = `# Routes Configuration
routes: []

`

const servicesConfig = `# Services Configuration
services:
  _defaults:
    autowire: true
    public: false
`

const indexPHP = `<?php
// GMCore Web Entry Point
echo "GMCore Framework";
`

const appYaml = `name: GMCore Application
version: 1.0.0
`
