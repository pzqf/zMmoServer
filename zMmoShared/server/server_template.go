package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// ServerTemplate 服务器启动模板
type ServerTemplate struct {
	Server       *BaseServer
	SetupFunc    func() error
	StartFunc    func() error
	ShutdownFunc func()
}

// NewServerTemplate 创建服务器启动模板
func NewServerTemplate(server *BaseServer) *ServerTemplate {
	return &ServerTemplate{
		Server: server,
	}
}

// SetSetupFunc 设置初始化函数
func (t *ServerTemplate) SetSetupFunc(setupFunc func() error) *ServerTemplate {
	t.SetupFunc = setupFunc
	return t
}

// SetStartFunc 设置启动函数
func (t *ServerTemplate) SetStartFunc(startFunc func() error) *ServerTemplate {
	t.StartFunc = startFunc
	return t
}

// SetShutdownFunc 设置关闭函数
func (t *ServerTemplate) SetShutdownFunc(shutdownFunc func()) *ServerTemplate {
	t.ShutdownFunc = shutdownFunc
	return t
}

// Run 运行服务器
func (t *ServerTemplate) Run() {
	// 第一步：使用默认配置初始化日志（只输出到控制台）
	defaultCfg := &zLog.Config{
		Level:    zLog.InfoLevel,
		Console:  true,
		Filename: "",
	}
	if err := zLog.InitLogger(defaultCfg); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}

	zLog.PrintLogo(t.Server.ServerName, t.Server.ServerVersion)

	// 第二步：执行初始化函数（加载配置文件）
	if t.SetupFunc != nil {
		zLog.Info("Setting up server...")
		if err := t.SetupFunc(); err != nil {
			zLog.Fatal("Failed to setup server", zap.Error(err))
		}
		zLog.Info("Server setup completed")
	}

	// 第三步：从配置文件重新初始化日志
	if logConfig := t.getLogConfig(); logConfig != nil {
		if err := zLog.InitLogger(logConfig); err != nil {
			zLog.Error("Failed to reinitialize logger with config, using default", zap.Error(err))
		} else {
			zLog.Info("Logger reinitialized with config")
		}
	}

	// 第四步：启动服务器
	zLog.Info(fmt.Sprintf("Starting %s Server...", t.Server.ServerType))
	if err := t.Server.Start(); err != nil {
		zLog.Fatal("Failed to start server", zap.Error(err))
	}

	// 第五步：执行启动后函数
	if t.StartFunc != nil {
		zLog.Info("Running post-start tasks...")
		if err := t.StartFunc(); err != nil {
			zLog.Error("Failed to run post-start tasks", zap.Error(err))
		}
		zLog.Info("Post-start tasks completed")
	}

	zLog.Info(fmt.Sprintf("%s Server started successfully!", t.Server.ServerType))
	zLog.Info("Server is running. Press Ctrl+C to exit.")

	// 第六步：等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	zLog.Info("Received shutdown signal, stopping server...")

	// 第七步：执行关闭函数
	if t.ShutdownFunc != nil {
		t.ShutdownFunc()
	}

	// 第八步：停止服务器
	t.Server.Stop()
	t.Server.Wait()

	zLog.Info(fmt.Sprintf("%s Server stopped gracefully", t.Server.ServerType))
}

// getLogConfig 从配置组件获取日志配置
func (t *ServerTemplate) getLogConfig() *zLog.Config {
	configComponent := t.Server.GetComponent("Config")
	if configComponent == nil {
		return nil
	}

	type LogConfigurable interface {
		GetLogConfig() *zLog.Config
	}

	if cfg, ok := configComponent.(LogConfigurable); ok {
		return cfg.GetLogConfig()
	}

	return nil
}
