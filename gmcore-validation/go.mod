module github.com/gmcorenet/sdk/gmcore-validation

go 1.23

require github.com/gmcorenet/framework v0.1.0

replace (
	github.com/gmcorenet/framework => ../../framework
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
)
