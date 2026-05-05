module github.com/gmcorenet/sdk/gmcore-i18n

go 1.21

require gopkg.in/yaml.v3 v3.0.1

replace (
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
	github.com/gmcorenet/sdk/gmcore-i18n => .
)
