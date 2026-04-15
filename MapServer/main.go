package main

import (
	"flag"
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/MapServer/config"
	"github.com/pzqf/zMmoServer/MapServer/server"
	"github.com/pzqf/zMmoServer/MapServer/version"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "config_single.ini", "Path to config file")
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

	zLog.PrintLogo("Map Server", version.Version)

	baseServer := server.NewBaseServer(cfg)
	baseServer.SetLogger(zLog.GetStandardLogger())

	if err := baseServer.Run(); err != nil {
		zLog.Fatal("Failed to start MapServer", zap.Error(err))
	}
}
