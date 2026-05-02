module github.com/gmcorenet/sdk/gmcore-uuid

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-uuid v0.0.0
	github.com/google/uuid v1.6.0
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-uuid => .
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)