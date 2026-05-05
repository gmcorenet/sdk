package gmcore_config

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
