module github.com/gmcorenet/sdk/gmcore-config

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-config v0.0.0
	gopkg.in/yaml.v3 v3.0.1
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-config => .
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)