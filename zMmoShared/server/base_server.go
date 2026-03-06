package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// ServerType 服务器类型
type ServerType string

const (
	ServerTypeGlobal  ServerType = "global"
	ServerTypeGateway ServerType = "gateway"
	ServerTypeGame    ServerType = "game"
	ServerTypeMap     ServerType = "map"
	ServerTypeAdmin   ServerType = "admin"
)

// BaseServer 基础服务器结构
type BaseServer struct {
	ServerType    ServerType
	ServerID      string
	ServerName    string
	ServerVersion string
	IsRunning     bool
	components    map[string]interface{}
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc

	// 生命周期钩子函数
	onBeforeStartFunc func() error
	onAfterStartFunc  func() error
	onBeforeStopFunc  func()
	onAfterStopFunc   func()
}

// NewBaseServer 创建基础服务器实例
func NewBaseServer(serverType ServerType, serverID, serverName, version string) *BaseServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &BaseServer{
		ServerType:    serverType,
		ServerID:      serverID,
		ServerName:    serverName,
		ServerVersion: version,
		IsRunning:     false,
		components:    make(map[string]interface{}),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start 启动服务器
func (s *BaseServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.IsRunning {
		return fmt.Errorf("server already running")
	}

	zLog.Info(fmt.Sprintf("Starting %s Server...", s.ServerType),
		zap.String("server_id", s.ServerID),
		zap.String("server_name", s.ServerName),
		zap.String("version", s.ServerVersion))

	// 启动前的准备工作
	if s.onBeforeStartFunc != nil {
		if err := s.onBeforeStartFunc(); err != nil {
			return err
		}
	}

	s.IsRunning = true

	// 启动后的工作
	if s.onAfterStartFunc != nil {
		if err := s.onAfterStartFunc(); err != nil {
			s.IsRunning = false
			return err
		}
	}

	zLog.Info(fmt.Sprintf("%s Server started successfully!", s.ServerType))
	return nil
}

// Stop 停止服务器
func (s *BaseServer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.IsRunning {
		return
	}

	zLog.Info(fmt.Sprintf("Stopping %s Server...", s.ServerType))

	// 停止前的工作
	if s.onBeforeStopFunc != nil {
		s.onBeforeStopFunc()
	}

	// 取消上下文
	s.cancel()

	s.IsRunning = false

	// 停止后的工作
	if s.onAfterStopFunc != nil {
		s.onAfterStopFunc()
	}

	zLog.Info(fmt.Sprintf("%s Server stopped gracefully", s.ServerType))
}

// Wait 等待服务器停止
func (s *BaseServer) Wait() {
	<-s.ctx.Done()
}

// RegisterComponent 注册组件
func (s *BaseServer) RegisterComponent(name string, component interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.components[name] = component
	zLog.Info("Component registered",
		zap.String("server_type", string(s.ServerType)),
		zap.String("component", name))
}

// GetComponent 获取组件
func (s *BaseServer) GetComponent(name string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.components[name]
}

// GetComponents 获取所有组件
func (s *BaseServer) GetComponents() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	components := make(map[string]interface{})
	for k, v := range s.components {
		components[k] = v
	}
	return components
}

// IsServerRunning 检查服务器是否运行
func (s *BaseServer) IsServerRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsRunning
}

// GetContext 获取服务器上下文
func (s *BaseServer) GetContext() context.Context {
	return s.ctx
}
