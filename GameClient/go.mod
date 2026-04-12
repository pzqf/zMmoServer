module github.com/pzqf/zMmoServer/GameClient

go 1.25.5

require (
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/pzqf/zCommon v0.0.0-00010101000000-000000000000
	github.com/pzqf/zEngine v0.0.2
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/golang/snappy v1.0.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/pzqf/zUtil v0.0.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

replace (
	github.com/pzqf/zCommon => ../zCommon
	github.com/pzqf/zEngine => ../../zEngine
	github.com/pzqf/zMmoServer => ../
	github.com/pzqf/zMmoServer/resources/protocol/net/protocol => ../resources/protocol/net/protocol
	github.com/pzqf/zUtil => ../../zUtil
)
