package main

import (
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GlobalServer/config"
	"github.com/pzqf/zMmoServer/GlobalServer/global"
	"github.com/pzqf/zMmoServer/GlobalServer/version"
)

func main() {
	// ========== 第一步：加载配置 ==========
	cfg, err := config.LoadConfig("config.ini")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// ========== 第二步：初始化日志 ==========
	if err := zLog.InitLogger(cfg.GetLogConfig()); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	// 打印服务器启动信息
	zLog.PrintLogo("Global Server", version.Version)

	// ========== 第三步：创建并运行服务器 ==========
	globalServer := global.NewBaseServer()

	// 注入日志器（必须）
	globalServer.BaseServer.SetLogger(zLog.GetStandardLogger())

	// 注入配置到子类
	globalServer.Config = cfg

	// 运行服务器（阻塞方法，直到收到退出信号）
	// 如果日志未初始化，会返回错误
	if err := globalServer.BaseServer.Run(); err != nil {
		fmt.Printf("Server run failed: %v\n", err)
	}
}
