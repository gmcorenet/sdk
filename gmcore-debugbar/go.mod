module github.com/gmcorenet/sdk/gmcore-debugbar

go 1.21

require (
	github.com/gmcorenet/framework v0.1.0
	github.com/gmcorenet/sdk/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/framework => ../../framework
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
)
