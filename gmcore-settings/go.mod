module github.com/gmcorenet/sdk/gmcore-settings

go 1.21

require (
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/gmcorenet/sdk/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
)