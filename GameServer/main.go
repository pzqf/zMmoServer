package main

import (
	"github.com/pzqf/zMmoServer/GameServer/game"
	"github.com/pzqf/zMmoShared/server"
)

func main() {
	// 创建游戏服基础服务器
	baseServer := game.NewBaseServer()

	// 创建服务器启动模板
	template := server.NewServerTemplate(baseServer.BaseServer)

	// 运行服务器
	template.Run()
}
