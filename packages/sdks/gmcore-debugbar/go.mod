module github.com/gmcorenet/sdk/gmcore-debugbar

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-debugbar v0.0.0
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-debugbar => .
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)