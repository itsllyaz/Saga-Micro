module github.com/itsllyaz/Saga-Micro

go 1.26.5

replace github.com/itsllyaz/saga-micro => ./

require (
	github.com/itsllyaz/saga-micro v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.21.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
)
