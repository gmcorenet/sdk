module github.com/gmcorenet/sdk/gmcore-validation

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-validation v0.0.0
	github.com/gmcorenet/sdk/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-validation => .
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
)