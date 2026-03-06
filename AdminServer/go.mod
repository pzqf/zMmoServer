module github.com/pzqf/zMmoServer/AdminServer

go 1.25.5

require (
	github.com/pzqf/zEngine v0.0.2
	go.uber.org/zap v1.27.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/pzqf/zUtil v0.0.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)

replace github.com/pzqf/zEngine => ../../zEngine

replace github.com/pzqf/zMmoShared => ../zMmoShared

replace github.com/pzqf/zUtil => ../../zUtil
