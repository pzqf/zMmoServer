package main

import (
	"flag"
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GatewayServer/config"
	"github.com/pzqf/zMmoServer/GatewayServer/gateway"
	"github.com/pzqf/zMmoServer/GatewayServer/version"
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

	if err := zLog.InitLogger(&cfg.Log); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	zLog.PrintLogo("Gateway Server", version.Version)

	gatewayServer := gateway.NewBaseServer(cfg)
	gatewayServer.SetLogger(zLog.GetStandardLogger())

	if err := gatewayServer.Run(); err != nil {
		zLog.Fatal("Failed to start GatewayServer", zap.Error(err))
	}
}
