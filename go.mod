module github.com/pzqf/zMmoServer

go 1.25

replace (
	github.com/pzqf/zCommon => ./zCommon
	github.com/pzqf/zEngine => ../zEngine
	github.com/pzqf/zMmoServer/resources/protocol/net/protocol => ./resources/protocol/net/protocol
)
