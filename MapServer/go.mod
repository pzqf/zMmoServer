module github.com/pzqf/zMmoServer/MapServer

go 1.25.5

require (
	github.com/pzqf/zEngine v0.0.2
	github.com/pzqf/zMmoServer v0.0.0-00010101000000-000000000000
	github.com/pzqf/zMmoShared v0.0.0-00010101000000-000000000000
	github.com/pzqf/zUtil v0.0.1
	go.uber.org/zap v1.27.0
	google.golang.org/protobuf v1.36.8
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/sys v0.40.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/pzqf/zEngine => ../../zEngine
	github.com/pzqf/zMmoServer => ..
	github.com/pzqf/zMmoShared => ../zMmoShared
	github.com/pzqf/zUtil => ../../zUtil
)
