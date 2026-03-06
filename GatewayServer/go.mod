module github.com/pzqf/zMmoServer/GatewayServer

go 1.25.5

require (
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/golang/snappy v1.0.0
	github.com/pzqf/zEngine v0.0.2
	github.com/pzqf/zMmoShared v0.0.0-00010101000000-000000000000
	github.com/pzqf/zUtil v0.0.1
	go.uber.org/zap v1.27.0
	google.golang.org/protobuf v1.36.8
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/pzqf/zEngine => ../../zEngine

replace github.com/pzqf/zMmoShared => ../zMmoShared

replace github.com/pzqf/zUtil => ../../zUtil
