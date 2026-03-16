package admin

import (
	"github.com/pzqf/zEngine/zServer"
	"github.com/pzqf/zMmoServer/AdminServer/config"
	"github.com/pzqf/zMmoServer/AdminServer/monitor"
)

// ServerType 管理服类型
const ServerTypeAdmin zServer.ServerType = "admin"

// BaseServer 管理服基础服务器
type BaseServer struct {
	*zServer.BaseServer
	Config         *config.Config
	MonitorService *monitor.MonitorService
}

// NewBaseServer 创建管理服基础服务器
func NewBaseServer() *BaseServer {
	// 先创建子类实例
	as := &BaseServer{}

	// 创建基础服务器，传入子类作为 hooks
	baseServer := zServer.NewBaseServer(
		ServerTypeAdmin,
		"admin-1",
		"Admin Server",
		"1.0.0",
		as, // 传入自身作为 LifecycleHooks 实现
	)

	as.BaseServer = baseServer
	return as
}

// OnBeforeStart 启动前的准备工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStart() error {
	cfg := s.Config
	if cfg == nil {
		return nil
	}

	// 初始化监控服务
	monitorService := monitor.NewMonitorService(cfg)
	s.MonitorService = monitorService

	// 注册组件
	s.RegisterComponent("Config", cfg)
	s.RegisterComponent("MonitorService", monitorService)

	return nil
}

// OnAfterStart 启动后的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnAfterStart() error {
	// 启动监控服务
	if err := s.MonitorService.Start(); err != nil {
		return err
	}

	return nil
}

// OnBeforeStop 停止前的工作 - 实现 LifecycleHooks 接口
func (s *BaseServer) OnBeforeStop() {
	// 停止监控服务
	if s.MonitorService != nil {
		s.MonitorService.Stop()
	}
}
