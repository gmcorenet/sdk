module github.com/gmcorenet/sdk/gmcore-lifecycle

go 1.21

require (
	gopkg.in/yaml.v3 v3.0.1
	github.com/gmcorenet/sdk/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
)