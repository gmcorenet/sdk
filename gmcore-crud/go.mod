module gmcore-crud

go 1.19

require (
	gmcore-form v0.0.0
	gmcore-settings v0.0.0
	gmcore-uuid v0.0.0
	gorm.io/gorm v1.25.10
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	gmcore-validation v0.0.0 // indirect
)

replace gmcore-form => ../gmcore-form

replace gmcore-debugbar => ../gmcore-debugbar

replace gmcore-settings => ../gmcore-settings

replace gmcore-uuid => ../gmcore-uuid

replace gmcore-validation => ../gmcore-validation
