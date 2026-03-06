package main

import (
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"github.com/pzqf/zMmoShared/server"
)

func main() {
	// 创建地图服基础服务器
	baseServer := maps.NewBaseServer()

	// 创建服务器启动模板
	template := server.NewServerTemplate(baseServer.BaseServer)

	// 运行服务器
	template.Run()
}
