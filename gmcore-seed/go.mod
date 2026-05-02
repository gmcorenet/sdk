module github.com/gmcorenet/sdk/gmcore-seed

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-seed v0.0.0
	github.com/gmcorenet/sdk/gmcore-uuid v0.0.0
	gorm.io/gorm v1.25.10
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-seed => .
	github.com/gmcorenet/sdk/gmcore-uuid => ../gmcore-uuid
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)