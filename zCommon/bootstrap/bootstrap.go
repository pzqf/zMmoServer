package bootstrap

import (
	"flag"
	"fmt"

	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zServer"
	"go.uber.org/zap"
)

type LogConfigProvider interface {
	GetLogConfig() *zLog.Config
}

type ServerConfig struct {
	Name       string
	Version    string
	ConfigPath string
}

func DefaultServerConfig(name, version string) ServerConfig {
	return ServerConfig{
		Name:       name,
		Version:    version,
		ConfigPath: "config.ini",
	}
}

type ServerFactory func() *zServer.BaseServer

func Run(cfg ServerConfig, configLoader func(path string) (LogConfigProvider, error), serverFactory ServerFactory) {
	flag.StringVar(&cfg.ConfigPath, "config", cfg.ConfigPath, "Path to config file")
	flag.Parse()

	logCfg, err := configLoader(cfg.ConfigPath)
	if err != nil {
		fmt.Printf("[%s] Failed to load config: %v\n", cfg.Name, err)
		return
	}

	if err := zLog.InitLogger(logCfg.GetLogConfig()); err != nil {
		fmt.Printf("[%s] Failed to initialize logger: %v\n", cfg.Name, err)
		return
	}

	zLog.PrintLogo(cfg.Name, cfg.Version)

	server := serverFactory()
	server.SetLogger(zLog.GetStandardLogger())

	if err := server.Run(); err != nil {
		zLog.Fatal("Server run failed",
			zap.String("server", cfg.Name),
			zap.Error(err))
	}
}
