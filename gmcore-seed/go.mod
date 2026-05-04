module github.com/gmcorenet/sdk/gmcore-seed

go 1.21

replace (
	github.com/gmcorenet/gmcore-error => ../gmcore-error
	github.com/gmcorenet/sdk/gmcore-seed => .
	github.com/gmcorenet/sdk/gmcore-uuid => ../gmcore-uuid
)
