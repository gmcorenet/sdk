module github.com/gmcorenet/sdk/gmcore-doctrine

go 1.21

require (
	github.com/gmcorenet/gmcore-error v0.1.0
	gorm.io/gorm v1.25.5
)

replace (
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)