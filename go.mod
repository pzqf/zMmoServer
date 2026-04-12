module github.com/pzqf/zMmoServer

go 1.25.5

replace (
	github.com/pzqf/zCommon => ./zCommon
	github.com/pzqf/zEngine => ../zEngine
	github.com/pzqf/zMmoServer => ./
	github.com/pzqf/zMmoServer/resources/protocol/net/protocol => ./resources/protocol/net/protocol
	github.com/pzqf/zUtil => ../zUtil
)
