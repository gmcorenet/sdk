module github.com/gmcorenet/sdk/gmcore-crud

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-form v0.0.0
	github.com/gmcorenet/sdk/gmcore-orm v0.0.0
	github.com/gmcorenet/sdk/gmcore-settings v0.0.0
	github.com/gmcorenet/sdk/gmcore-uid v0.0.0
	gorm.io/gorm v1.25.10
)

require (
	github.com/gmcorenet/sdk/gmcore-validation v0.0.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
)

replace (
	github.com/gmcorenet/gmcore-error => ../gmcore-error
	github.com/gmcorenet/sdk/gmcore-crud => .
	github.com/gmcorenet/sdk/gmcore-form => ../gmcore-form
	github.com/gmcorenet/sdk/gmcore-orm => ../gmcore-orm
	github.com/gmcorenet/sdk/gmcore-settings => ../gmcore-settings
	github.com/gmcorenet/sdk/gmcore-uid => ../gmcore-uid
	github.com/gmcorenet/sdk/gmcore-validation => ../gmcore-validation
)
