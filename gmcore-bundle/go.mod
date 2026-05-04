module github.com/gmcorenet/sdk/gmcore-bundle

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-installer v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

replace (
	github.com/gmcorenet/gmcore-error => ../gmcore-error
	github.com/gmcorenet/sdk/gmcore-bundle => .
	github.com/gmcorenet/sdk/gmcore-installer => ../gmcore-installer
)
