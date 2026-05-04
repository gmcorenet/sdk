module github.com/gmcorenet/sdk/gmcore-crud

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-crud v0.0.0
	github.com/gmcorenet/sdk/gmcore-form v0.0.0
	github.com/gmcorenet/sdk/gmcore-settings v0.0.0
	github.com/gmcorenet/sdk/gmcore-uuid v0.0.0
	github.com/gmcorenet/sdk/gmcore-validation v0.0.0
	gorm.io/gorm v1.25.10
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-crud => .
	github.com/gmcorenet/sdk/gmcore-form => ../gmcore-form
	github.com/gmcorenet/sdk/gmcore-settings => ../gmcore-settings
	github.com/gmcorenet/sdk/gmcore-uuid => ../gmcore-uuid
	github.com/gmcorenet/sdk/gmcore-validation => ../gmcore-validation
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)