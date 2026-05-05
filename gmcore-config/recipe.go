package gmcore_config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Recipe struct {
	Name         string
	Version      string
	ConfigFiles  []ConfigFile
	Dependencies []string
}

type ConfigFile struct {
	Path    string
	Content string
	Mode    os.FileMode
}

var AllRecipes = []Recipe{
	{
		Name:    "gmcore-transport",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/transport.yaml",
				Content: `server:
  mode: uds
  uds:
    path: var/socket/app.sock
    perm: 0660
    group: gmcore
    auto_remove: false
  tcp:
    host: 0.0.0.0
    ports: [8080]

security:
  type: hmac
  key: %env(TRANSPORT_SECRET)%
`,
				Mode: 0644,
			},
		},
	},
	{
		Name:    "gmcore-log",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/log.yaml",
				Content: `level: info

handlers:
  - type: console
    params:
      format: text

  - type: rotating
    params:
      filename: var/log/app.log
      max_size: 10485760
      max_backups: 5
      format: json
`,
				Mode: 0644,
			},
		},
	},
	{
		Name:    "gmcore-router",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/routes.yaml",
				Content: `routes:
  home:
    path: /
    handler: HomeController.Index
    methods: [GET]
`,
				Mode: 0644,
			},
		},
	},
	{
		Name:    "gmcore-cache",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/cache.yaml",
				Content: `adapter: memory
ttl: 3600
prefix: app_
`,
				Mode: 0644,
			},
		},
	},
	{
		Name:    "gmcore-security",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/security.yaml",
				Content: `role_prefix: "ROLE_"
default_role: "ROLE_USER"
password_cost: 10

firewall:
  enabled: true
  patterns:
    - ^/admin
  excludes:
    - ^/health
`,
				Mode: 0644,
			},
		},
	},
	{
		Name:    "gmcore-session",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/session.yaml",
				Content: `name: gmcore_session
lifetime: 3600
path: /
secure: true
http_only: true
same_site: strict
`,
				Mode: 0644,
			},
		},
	},
	{
		Name:    "gmcore-mailer",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/mailer.yaml",
				Content: `host: %env(SMTP_HOST)%
port: 587
username: %env(SMTP_USER)%
password: %env(SMTP_PASS)%
from: %env(MAILER_FROM)%
from_name: My App
encryption: tls
`,
				Mode: 0600,
			},
		},
	},
	{
		Name:    "gmcore-i18n",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/i18n.yaml",
				Content: `default_locale: en
fallback_locale: en

directories:
  - translations/en
`,
				Mode: 0644,
			},
		},
	},
	{
		Name:    "gmcore-events",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/events.yaml",
				Content: `listeners: {}
`,
				Mode: 0644,
			},
		},
	},
	{
		Name:    "gmcore-messenger",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/messenger.yaml",
				Content: `worker_count: 4

retry_policy:
  max_retries: 3
  initial_delay: 1000
  max_delay: 60000
  multiplier: 2.0

transport: memory
`,
				Mode: 0644,
			},
		},
	},
	{
		Name:    "gmcore-orm",
		Version: "1.0.0",
		ConfigFiles: []ConfigFile{
			{
				Path: "config/database.yaml",
				Content: `driver: mysql
dsn: %env(DATABASE_DSN)%

pool:
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 3600
  conn_max_idle_time: 600

auto_migrate: true

logging:
  level: info
  slow_threshold: 1000
`,
				Mode: 0600,
			},
		},
	},
}

type RecipeRegistry struct {
	enabled map[string]bool
}

func NewRecipeRegistry() *RecipeRegistry {
	enabled := make(map[string]bool)
	for _, r := range AllRecipes {
		enabled[r.Name] = true
	}
	return &RecipeRegistry{enabled: enabled}
}

func (r *RecipeRegistry) Enable(name string) {
	r.enabled[name] = true
}

func (r *RecipeRegistry) Disable(name string) {
	delete(r.enabled, name)
}

func (r *RecipeRegistry) GetEnabledRecipes() []Recipe {
	var recipes []Recipe
	for _, recipe := range AllRecipes {
		if r.enabled[recipe.Name] {
			recipes = append(recipes, recipe)
		}
	}
	return recipes
}

func (r *RecipeRegistry) RenderRecipe(recipe Recipe, vars map[string]string) ([]ConfigFile, error) {
	var result []ConfigFile

	for _, cf := range recipe.ConfigFiles {
		tmpl, err := template.New(cf.Path).Parse(cf.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template for %s: %w", cf.Path, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vars); err != nil {
			return nil, fmt.Errorf("failed to render %s: %w", cf.Path, err)
		}

		result = append(result, ConfigFile{
			Path:    cf.Path,
			Content: buf.String(),
			Mode:    cf.Mode,
		})
	}

	return result, nil
}

func (r *RecipeRegistry) InstallRecipes(appPath string, vars map[string]string, force bool) error {
	recipes := r.GetEnabledRecipes()

	for _, recipe := range recipes {
		files, err := r.RenderRecipe(recipe, vars)
		if err != nil {
			return fmt.Errorf("failed to render recipe %s: %w", recipe.Name, err)
		}

		for _, file := range files {
			fullPath := filepath.Join(appPath, file.Path)

			exists := false
			if _, err := os.Stat(fullPath); err == nil {
				exists = true
			}

			if exists && !force {
				merged, err := r.mergeConfig(fullPath, file.Content)
				if err != nil {
					return fmt.Errorf("failed to merge config %s: %w", file.Path, err)
				}
				file.Content = merged
			}

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

func (r *RecipeRegistry) mergeConfig(existingPath, newContent string) (string, error) {
	existingData, err := os.ReadFile(existingPath)
	if err != nil {
		return newContent, nil
	}

	var existing, news, merged map[string]interface{}

	if err := yaml.Unmarshal(existingData, &existing); err != nil {
		return newContent, nil
	}

	if err := yaml.Unmarshal([]byte(newContent), &news); err != nil {
		return newContent, nil
	}

	merged = r.mergeMap(existing, news)

	out, err := yaml.Marshal(merged)
	if err != nil {
		return newContent, err
	}

	return string(out), nil
}

func (r *RecipeRegistry) mergeMap(existing, new map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range existing {
		result[k] = v
	}

	for k, v := range new {
		if ev, ok := existing[k]; ok {
			if em, ok := ev.(map[string]interface{}); ok {
				if nm, ok := v.(map[string]interface{}); ok {
					result[k] = r.mergeMap(em, nm)
					continue
				}
			}
		}
		result[k] = v
	}

	return result
}

func (r *RecipeRegistry) GenerateManifest() string {
	var buf bytes.Buffer
	buf.WriteString("# GMCore SDK Recipes\n\n")
	buf.WriteString("| SDK | Version | Config Files |\n")
	buf.WriteString("|-----|---------|-------------|\n")

	for _, recipe := range AllRecipes {
		buf.WriteString(fmt.Sprintf("| %s | %s | ", recipe.Name, recipe.Version))
		for i, cf := range recipe.ConfigFiles {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(cf.Path)
		}
		buf.WriteString(" |\n")
	}

	return buf.String()
}
