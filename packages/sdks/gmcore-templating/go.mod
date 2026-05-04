module github.com/gmcorenet/sdk/gmcore-templating

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-templating v0.0.0
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-templating => .
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)