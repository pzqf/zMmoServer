package main

import (
	"flag"
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GlobalServer/config"
	"github.com/pzqf/zMmoServer/GlobalServer/global"
	"github.com/pzqf/zMmoServer/GlobalServer/version"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "config.ini", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	if err := zLog.InitLogger(cfg.GetLogConfig()); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	zLog.PrintLogo("Global Server", version.Version)

	globalServer := global.NewBaseServer(cfg)
	globalServer.SetLogger(zLog.GetStandardLogger())

	if err := globalServer.Run(); err != nil {
		zLog.Fatal("Failed to start GlobalServer", zap.Error(err))
	}
}
