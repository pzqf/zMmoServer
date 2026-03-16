module github.com/pzqf/zMmoServer/testclient

go 1.25

require (
	github.com/pzqf/zEngine v0.0.0-00010101000000-000000000000
	github.com/pzqf/zMmoShared v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.33.0
)

replace (
	github.com/pzqf/zEngine => ../../zEngine
	github.com/pzqf/zMmoShared => ../zMmoShared
)