module github.com/gmcorenet/sdk/gmcore-security

go 1.23

require (
	github.com/gmcorenet/framework v0.1.0
	github.com/gmcorenet/sdk/gmcore-config v0.1.0
	golang.org/x/crypto v0.28.0
)

require (
	github.com/gmcorenet/sdk/gmcore-error v0.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/gmcorenet/framework => ../../framework
	github.com/gmcorenet/sdk/gmcore-config => ../gmcore-config
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
)
