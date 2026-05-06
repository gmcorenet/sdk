module github.com/gmcorenet/sdk/gmcore-view

go 1.23

require (
	github.com/gmcorenet/sdk/gmcore-i18n v0.1.0
	github.com/gmcorenet/sdk/gmcore-templating v0.1.0
)

require (
	github.com/kr/text v0.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
	github.com/gmcorenet/sdk/gmcore-i18n => ../gmcore-i18n
	github.com/gmcorenet/sdk/gmcore-templating => ../gmcore-templating
	github.com/gmcorenet/sdk/gmcore-view => .
)
