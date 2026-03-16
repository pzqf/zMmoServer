module github.com/pzqf/zMmoServer/AdminServer

go 1.25.5

require (
	github.com/pzqf/zEngine v0.0.2
	github.com/pzqf/zUtil v0.0.1
	go.uber.org/zap v1.27.0
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/pzqf/zEngine => ../../zEngine

replace github.com/pzqf/zMmoShared => ../zMmoShared

replace github.com/pzqf/zUtil => ../../zUtil
