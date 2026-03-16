package main

import (
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/maps"
	"github.com/pzqf/zMmoServer/MapServer/version"
	"go.uber.org/zap"
)

func main() {
	// 打印服务器启动信息
	zLog.PrintLogo("Map Server", version.Version)

	// 打印版本详细信息
	verInfo := version.GetVersion()
	zLog.Info("Map Server Version Info",
		zap.String("version", verInfo["version"]),
		zap.String("build_time", verInfo["build_time"]),
		zap.String("git_commit", verInfo["git_commit"]),
		zap.String("go_version", verInfo["go_version"]),
		zap.String("os", verInfo["os"]),
		zap.String("arch", verInfo["arch"]),
	)

	// 创建地图服基础服务器
	baseServer := maps.NewBaseServer()

	// 启动服务器
	if err := baseServer.Start(); err != nil {
		zLog.Fatal("Failed to start MapServer", zap.Error(err))
	}

	// 等待服务器运行
	zLog.Info("MapServer started successfully")
	// 等待信号退出
	select {}
}
