module github.com/pzqf/zMmoServer

go 1.25

require (
	github.com/gorilla/websocket v1.5.1
	github.com/pzqf/zEngine v0.0.0-00010101000000-000000000000
	github.com/pzqf/zMmoShared v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.26.0
	google.golang.org/protobuf v1.31.0
)

replace (
	github.com/pzqf/zEngine => ../zEngine
	github.com/pzqf/zMmoShared => ./zMmoShared
	github.com/pzqf/zMmoServer/resources/protocol/net/protocol => ./resources/protocol/net/protocol
)