module github.com/gmcorenet/sdk/gmcore-ratelimit

go 1.24

require (
	github.com/bradfitz/gomemcache v0.0.0-20260422231931-4d751bb6e37c
	github.com/gmcorenet/framework v0.1.0
	github.com/redis/go-redis/v9 v9.19.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/stretchr/testify v1.8.4 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)

replace (
	github.com/gmcorenet/framework => ../../framework
	github.com/gmcorenet/sdk/gmcore-error => ../gmcore-error
	github.com/gmcorenet/sdk/gmcore-ratelimit => .
)
