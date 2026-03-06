package global

import (
	"github.com/pzqf/zMmoServer/GlobalServer/config"
	"github.com/pzqf/zMmoServer/GlobalServer/http"
	"github.com/pzqf/zMmoShared/server"
)

// BaseServer 全局服基础服务器
type BaseServer struct {
	*server.BaseServer
	Config     *config.Config
	HTTPService *http.Service
}

// NewBaseServer 创建全局服基础服务器
func NewBaseServer() *BaseServer {
	baseServer := server.NewBaseServer(
		server.ServerTypeGlobal,
		"global-1",
		"Global Server",
		"1.0.0",
	)

	server := &BaseServer{
		BaseServer: baseServer,
	}

	// 设置生命周期钩子
	server.onBeforeStartFunc = server.onBeforeStart
	server.onAfterStartFunc = server.onAfterStart
	server.onBeforeStopFunc = server.onBeforeStop

	return server
}

// onBeforeStart 启动前的准备工作
func (s *BaseServer) onBeforeStart() error {
	// 加载配置
	cfg, err := config.LoadConfig("config.ini")
	if err != nil {
		return err
	}
	s.Config = cfg

	// 创建HTTP服务
	s.HTTPService = http.NewService()
	s.HTTPService.SetConfig(cfg)

	// 初始化HTTP服务
	if err := s.HTTPService.Init(); err != nil {
		return err
	}

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("HTTPService", s.HTTPService)

	return nil
}

// onAfterStart 启动后的工作
func (s *BaseServer) onAfterStart() error {
	// 启动HTTP服务
	if err := s.HTTPService.Start(); err != nil {
		return err
	}

	return nil
}

// onBeforeStop 停止前的工作
func (s *BaseServer) onBeforeStop() {
	// 停止HTTP服务
	if s.HTTPService != nil {
		s.HTTPService.Stop()
	}
}
