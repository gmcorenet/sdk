module github.com/gmcorenet/sdk/gmcore-cert

go 1.21

require (
	github.com/gmcorenet/sdk/gmcore-cert v0.0.0
	golang.org/x/crypto v0.17.0
	github.com/gmcorenet/gmcore-error v0.1.0
)

replace (
	github.com/gmcorenet/sdk/gmcore-cert => .
	github.com/gmcorenet/gmcore-error => ../gmcore-error
)