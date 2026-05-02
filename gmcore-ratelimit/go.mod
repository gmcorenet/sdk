module github.com/gmcorenet/sdk/gmcore-ratelimit

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-ratelimit v0.0.0
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-ratelimit => .
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)