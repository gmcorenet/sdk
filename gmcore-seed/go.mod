module github.com/gmcorenet/sdk/gmcore-seed

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-uid v0.0.0
	gorm.io/gorm v1.25.10
	github.com/gmcorenet/sdk/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-uid => ../gmcore-uid
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
)