module github.com/gmcorenet/sdk/gmcore-cert

go 1.21

require golang.org/x/crypto v0.28.0

require (
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/text v0.19.0 // indirect
)

replace (
	github.com/gmcorenet/sdk/gmcore-cert => .
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
)
