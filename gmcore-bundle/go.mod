module github.com/gmcorenet/sdk/gmcore-bundle

go 1.21

require gopkg.in/yaml.v3 v3.0.1

replace (
	github.com/gmcorenet/gmcore-error => ../gmcore-error
	github.com/gmcorenet/sdk/gmcore-bundle => .
)
