module github.com/gmcorenet/sdk/gmcore-config

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-error v0.1.0
	gopkg.in/yaml.v3 v3.0.1
)

replace (
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
	github.com/gmcorenet/sdk/gmcore-config => .
)
