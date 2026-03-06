package main

import (
	"github.com/pzqf/zMmoServer/GatewayServer/gateway"
	"github.com/pzqf/zMmoShared/server"
)

func main() {
	// 创建网关服基础服务器
	baseServer := gateway.NewBaseServer()

	// 创建服务器启动模板
	template := server.NewServerTemplate(baseServer.BaseServer)

	// 运行服务器
	template.Run()
}
