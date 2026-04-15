package main

import (
	"flag"
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zMmoServer/GameServer/config"
	"github.com/pzqf/zMmoServer/GameServer/game"
	"github.com/pzqf/zMmoServer/GameServer/version"
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

	zLog.PrintLogo("Game Server", version.Version)

	gameServer := game.NewBaseServer(cfg)
	gameServer.SetLogger(zLog.GetStandardLogger())

	if err := gameServer.Run(); err != nil {
		zLog.Fatal("Failed to start GameServer", zap.Error(err))
	}
}
