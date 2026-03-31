package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zService"
	"github.com/pzqf/zMmoServer/GlobalServer/config"
	"github.com/pzqf/zMmoServer/GlobalServer/handler"
	"github.com/pzqf/zMmoServer/GlobalServer/metrics"
	"github.com/pzqf/zMmoServer/GlobalServer/version"
	"go.uber.org/zap"
)

// ShutdownFunc 关闭回调函数类型
type ShutdownFunc func()

// HttpService Echo-based HTTP service
type HttpService struct {
	zService.BaseService
	echo         *echo.Echo
	httpCfg      *config.HTTPConfig
	shutdownFunc ShutdownFunc // 关闭回调函数
	metrics      *metrics.Metrics
}

// Config returns the HTTP service configuration
func (s *HttpService) Config() *config.HTTPConfig {
	return s.httpCfg
}

// NewService creates a new Echo HTTP service
func NewService() *HttpService {
	e := echo.New()

	// 关闭 Echo 的 banner 输出
	e.HideBanner = true

	// Configure middleware - 使用自定义日志中间件
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","method":"${method}","uri":"${uri}","status":${status},"latency":${latency},"ip":"${remote_ip}"}` + "\n",
		Output: zLog.GetStandardLogger().Writer(),
	}))
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20))) // 20 requests per second

	service := &HttpService{
		BaseService: *zService.NewBaseService("http_service"),
		echo:        e,
	}

	return service
}

// SetConfig 设置配置（必须在 Init 之前调用）
func (s *HttpService) SetConfig(cfg *config.HTTPConfig) {
	s.httpCfg = cfg
}

// SetShutdownFunc 设置关闭回调函数
func (s *HttpService) SetShutdownFunc(fn ShutdownFunc) {
	s.shutdownFunc = fn
}

// SetMetrics 设置 metrics 实例
func (s *HttpService) SetMetrics(m *metrics.Metrics) {
	s.metrics = m
}

// Init initializes the HTTP service
func (s *HttpService) Init() error {
	if s.httpCfg == nil {
		return nil
	}

	s.SetState(zService.ServiceStateInit)
	zLog.Info("Initializing Echo HTTP service...", zap.String("listen_address", s.httpCfg.ListenAddress))

	// 添加 metrics 中间件
	if s.metrics != nil {
		s.echo.Use(s.metricsMiddleware())
	}

	// Register routes
	s.registerRoutes()

	return nil
}

// metricsMiddleware metrics 记录中间件
func (s *HttpService) metricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 将 metrics 存入上下文
			c.Set("metrics", s.metrics)

			start := time.Now()
			err := next(c)
			duration := time.Since(start)

			// 记录 HTTP 请求指标
			s.metrics.IncrementHTTPRequests()
			s.metrics.RecordHTTPRequest(duration)

			// 记录 HTTP 错误请求
			if err != nil {
				s.metrics.IncrementHTTPErrorRequests()
			}

			return err
		}
	}
}

// Start starts the HTTP service
func (s *HttpService) Start() error {
	if s.httpCfg == nil {
		zLog.Info("HTTP service config not set, skipping start")
		return nil
	}

	s.SetState(zService.ServiceStateRunning)
	zLog.Info("Starting Echo HTTP service...", zap.String("listen_address", s.httpCfg.ListenAddress))

	// 尝试启动服务器，检查端口是否可用
	ln, err := net.Listen("tcp", s.httpCfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.httpCfg.ListenAddress, err)
	}
	ln.Close()

	// Start server in a goroutine
	go func() {
		if err := s.echo.Start(s.httpCfg.ListenAddress); err != nil && err != http.ErrServerClosed {
			zLog.Error("HTTP service error", zap.Error(err))
		}
	}()

	zLog.Info("Echo HTTP service started successfully")
	return nil
}

// Stop stops the HTTP service
func (s *HttpService) Stop() error {
	if s.GetState() != zService.ServiceStateRunning {
		return nil
	}

	s.SetState(zService.ServiceStateStopping)
	zLog.Info("Stopping Echo HTTP service...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server
	if err := s.echo.Shutdown(ctx); err != nil {
		zLog.Error("Failed to stop HTTP service", zap.Error(err))
		return err
	}

	s.SetState(zService.ServiceStateStopped)
	zLog.Info("Echo HTTP service stopped")
	return nil
}

// Serve implements the Service interface
func (s *HttpService) Serve() {
	if s.httpCfg == nil {
		return
	}

	// Start the HTTP server
	if err := s.Start(); err != nil {
		zLog.Error("Failed to serve HTTP service", zap.Error(err))
		return
	}

	// Wait for server to stop
	//select {}
}

// registerRoutes registers all API routes
func (s *HttpService) registerRoutes() {
	// Health check
	s.echo.GET("/health", s.handleHealthCheck)

	// Shutdown endpoint for testing graceful shutdown
	s.echo.POST("/shutdown", s.handleShutdown)

	// API v1 routes
	api := s.echo.Group("/api/v1")

	// Account routes
	account := api.Group("/account")
	{
		account.POST("/create", handler.HandleAccountCreate)
		account.POST("/login", handler.HandleAccountLogin)
	}

	// Server routes
	server := api.Group("/server")
	{
		server.GET("/list", handler.HandleGetServerList)
		server.GET("/group/:groupId", handler.HandleGetServerListByGroup)
	}
}

// handleHealthCheck handles health check requests
func (s *HttpService) handleHealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"service":    "GlobalServer",
		"version":    version.Version,
		"build_time": version.BuildTime,
		"git_commit": version.GitCommit,
		"time":       time.Now().Format(time.RFC3339),
	})
}

// handleShutdown handles shutdown requests for testing graceful shutdown
func (s *HttpService) handleShutdown(c echo.Context) error {
	zLog.Info("Shutdown requested via HTTP API")

	// 触发关闭回调
	if s.shutdownFunc != nil {
		go s.shutdownFunc() // 在 goroutine 中执行，避免阻塞 HTTP 响应
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "shutting_down",
		"message": "Server is shutting down gracefully",
	})
}
