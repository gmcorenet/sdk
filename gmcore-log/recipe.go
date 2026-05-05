package gmcore_log

import "github.com/gmcorenet/sdk/gmcore-config"

type LogRecipeProvider struct{}

func (p *LogRecipeProvider) Recipes() []gmcore_config.Recipe {
	return []gmcore_config.Recipe{
		{
			Name:    "gmcore-log",
			Version: "1.0.0",
			ConfigFiles: []gmcore_config.ConfigFile{
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
			Dependencies: []string{},
		},
	}
}
