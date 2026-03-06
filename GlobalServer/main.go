package main

import (
	"github.com/pzqf/zMmoServer/GlobalServer/global"
	"github.com/pzqf/zMmoShared/server"
)

func main() {
	// 创建全局服基础服务器
	baseServer := global.NewBaseServer()

	// 创建服务器启动模板
	template := server.NewServerTemplate(baseServer.BaseServer)

	// 运行服务器
	template.Run()
}
