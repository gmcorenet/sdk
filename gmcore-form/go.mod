module github.com/gmcorenet/sdk/gmcore-form

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-form v0.0.0
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-form => .
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)