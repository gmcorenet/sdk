module github.com/gmcorenet/sdk/gmcore-response

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
)