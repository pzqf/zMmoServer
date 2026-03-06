package main

import (
	"github.com/pzqf/zMmoServer/AdminServer/admin"
	"github.com/pzqf/zMmoShared/server"
)

func main() {
	// 创建管理服基础服务器
	baseServer := admin.NewBaseServer()

	// 创建服务器启动模板
	template := server.NewServerTemplate(baseServer.BaseServer)

	// 运行服务器
	template.Run()
}
