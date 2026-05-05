package gmcore_transport

import "github.com/gmcorenet/sdk/gmcore-config"

type TransportRecipeProvider struct{}

func (p *TransportRecipeProvider) Recipes() []gmcore_config.Recipe {
	return []gmcore_config.Recipe{
		{
			Name:    "gmcore-transport",
			Version: "1.0.0",
			ConfigFiles: []gmcore_config.ConfigFile{
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
			Dependencies: []string{},
		},
	}
}
