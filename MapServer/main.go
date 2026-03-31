package main

import (
	"flag"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/server"
	"github.com/pzqf/zMmoServer/MapServer/version"
	"go.uber.org/zap"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "config_single.ini", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		zLog.Fatal("Failed to load config", zap.Error(err))
	}
	if err := zLog.InitLogger(cfg.GetLogConfig()); err != nil {
		zLog.Fatal("Failed to initialize logger", zap.Error(err))
	}

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
	baseServer := server.NewBaseServer(*configPath)
	baseServer.SetLogger(zLog.GetStandardLogger())

	// 启动服务器
	if err := baseServer.Run(); err != nil {
		zLog.Fatal("Failed to start MapServer", zap.Error(err))
	}
}
