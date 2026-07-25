package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
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
	// OPT-4: 不再用 middleware.CORS()（默认放行所有来源 *，任意网站都能跨域打登录 API）。
	// 仅当经 env ZMMO_HTTP_ALLOW_ORIGINS(逗号分隔)显式配置白名单时才放行对应来源；未配置则不加
	// CORS 中间件——不下发 Access-Control-Allow-Origin，浏览器按同源策略拒绝跨域；原生客户端不发
	// Origin 不受影响。
	if v := strings.TrimSpace(os.Getenv("ZMMO_HTTP_ALLOW_ORIGINS")); v != "" {
		origins := make([]string, 0)
		for _, o := range strings.Split(v, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
		if len(origins) > 0 {
			e.Use(middleware.CORSWithConfig(middleware.CORSConfig{AllowOrigins: origins}))
			zLog.Info("HTTP CORS restricted to configured origins", zap.Strings("origins", origins))
		}
	}
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
	// SEC-3：/shutdown 仅允许**本机**触发，拒绝远程——否则任何能访问 8888 端口的人可单包关停
	// 全局服（账号/登录/服务器列表的唯一入口）。用真实 TCP peer RemoteAddr 判断 loopback，
	// 不用 c.RealIP()（后者受 X-Forwarded-For 欺骗）。
	host, _, err := net.SplitHostPort(c.Request().RemoteAddr)
	if err != nil {
		host = c.Request().RemoteAddr
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		zLog.Warn("Rejected non-local shutdown request", zap.String("remote", c.Request().RemoteAddr))
		return c.JSON(http.StatusForbidden, map[string]string{"error": "shutdown allowed from localhost only"})
	}

	zLog.Info("Shutdown requested via HTTP API (localhost)")
	if s.shutdownFunc != nil {
		go s.shutdownFunc() // 在 goroutine 中执行，避免阻塞 HTTP 响应
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status":  "shutting_down",
		"message": "Server is shutting down gracefully",
	})
}
