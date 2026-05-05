module github.com/gmcorenet/sdk/gmcore-orm

go 1.21

require gorm.io/gorm v1.25.10

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
)

replace github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error

replace github.com/gmcorenet/sdk/gmcore-uid => ../gmcore-uid
