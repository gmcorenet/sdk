module github.com/gmcorenet/sdk/gmcore-crud

go 1.23

require (
	github.com/gmcorenet/sdk/gmcore-form v0.0.0
	github.com/gmcorenet/sdk/gmcore-orm v0.0.0
	github.com/gmcorenet/sdk/gmcore-settings v0.0.0
	github.com/gmcorenet/sdk/gmcore-uid v0.0.0
	gorm.io/gorm v1.25.10
)

require (
	github.com/gmcorenet/sdk/gmcore-config v0.0.0-20260505162658-8049bbe79a86 // indirect
	github.com/gmcorenet/sdk/gmcore-error v0.1.0 // indirect
	github.com/gmcorenet/sdk/gmcore-validation v0.0.0 // indirect
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/microsoft/go-mssqldb v1.6.0 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/mysql v1.5.7 // indirect
	gorm.io/driver/postgres v1.5.9 // indirect
	gorm.io/driver/sqlite v1.5.7 // indirect
	gorm.io/driver/sqlserver v1.5.3 // indirect
)

replace (
	github.com/gmcorenet/sdk/gmcore-crud => .
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
	github.com/gmcorenet/sdk/gmcore-form => ../gmcore-form
	github.com/gmcorenet/sdk/gmcore-orm => ../gmcore-orm
	github.com/gmcorenet/sdk/gmcore-settings => ../gmcore-settings
	github.com/gmcorenet/sdk/gmcore-uid => ../gmcore-uid
	github.com/gmcorenet/sdk/gmcore-validation => ../gmcore-validation
)
