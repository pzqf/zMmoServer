package main

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/game"
	"github.com/pzqf/zMmoServer/GameServer/version"
	"go.uber.org/zap"
)

func main() {
	// ========== 第一步：加载配置 ==========
	cfg, err := config.LoadConfig("config.ini")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// 打印配置信息
	fmt.Printf("GatewayAddr: %s\n", cfg.Gateway.GatewayAddr)
	fmt.Printf("ListenAddr: %s\n", cfg.Server.ListenAddr)

	// ========== 第二步：初始化日志 ==========
	if err := zLog.InitLogger(cfg.GetLogConfig()); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	// 打印服务器启动信息
	zLog.PrintLogo("Game Server", version.Version)

	// 打印版本详细信息
	verInfo := version.GetVersion()
	zLog.Info("Game Server Version Info",
		zap.String("version", verInfo["version"]),
		zap.String("build_time", verInfo["build_time"]),
		zap.String("git_commit", verInfo["git_commit"]),
		zap.String("go_version", verInfo["go_version"]),
		zap.String("os", verInfo["os"]),
		zap.String("arch", verInfo["arch"]),
	)

	// 打印配置信息到日志
	zLog.Info("Config loaded")
	fmt.Printf("GatewayAddr: %s\n", cfg.Gateway.GatewayAddr)
	fmt.Printf("ListenAddr: %s\n", cfg.Server.ListenAddr)

	// ========== 第三步：创建并运行服务器 ==========
	gameServer := game.NewBaseServer()

	// 注入日志器（必须）
	gameServer.BaseServer.SetLogger(zLog.GetStandardLogger())

	// 注入配置到子类
	gameServer.Config = cfg

	// 运行服务器（阻塞方法，直到收到退出信号）
	// 如果日志未初始化，会返回错误
	if err := gameServer.BaseServer.Run(); err != nil {
		fmt.Printf("Server run failed: %v\n", err)
	}
}
