package admin

import (
	"github.com/pzqf/zMmoServer/AdminServer/config"
	"github.com/pzqf/zMmoServer/AdminServer/monitor"
	"github.com/pzqf/zMmoShared/server"
)

// BaseServer 管理服基础服务器
type BaseServer struct {
	*server.BaseServer
	Config         *config.Config
	MonitorService *monitor.MonitorService
}

// NewBaseServer 创建管理服基础服务器
func NewBaseServer() *BaseServer {
	baseServer := server.NewBaseServer(
		server.ServerTypeAdmin,
		"admin-1",
		"Admin Server",
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

	// 初始化监控服务
	monitorService, err := monitor.NewMonitorService(cfg)
	if err != nil {
		return err
	}
	s.MonitorService = monitorService

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("MonitorService", monitorService)

	return nil
}

// onAfterStart 启动后的工作
func (s *BaseServer) onAfterStart() error {
	// 启动监控服务
	if err := s.MonitorService.Start(); err != nil {
		return err
	}

	return nil
}

// onBeforeStop 停止前的工作
func (s *BaseServer) onBeforeStop() {
	// 停止监控服务
	if s.MonitorService != nil {
		s.MonitorService.Stop()
	}
}
