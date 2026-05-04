module github.com/gmcorenet/sdk/gmcore-resolver

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-resolver v0.0.0
	gopkg.in/yaml.v3 v3.0.1
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-resolver => .
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)