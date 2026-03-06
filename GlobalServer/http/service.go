package http

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zService"
	"github.com/pzqf/zMmoServer/GlobalServer/config"
	"github.com/pzqf/zMmoServer/GlobalServer/handler"
	"go.uber.org/zap"
)

// Service Echo-based HTTP service
type Service struct {
	zService.BaseService
	echo      *echo.Echo
	cfg       *config.Config
	isRunning bool
}

// Config returns the HTTP service configuration
func (s *Service) Config() *config.HTTPConfig {
	return &s.cfg.HTTP
}

// NewService creates a new Echo HTTP service
func NewService() *Service {
	e := echo.New()

	// Configure middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20))) // 20 requests per second

	service := &Service{
		BaseService: *zService.NewBaseService("http_service"),
		echo:        e,
		isRunning:   false,
	}

	return service
}

// SetConfig 设置配置（必须在 Init 之前调用）
func (s *Service) SetConfig(cfg *config.Config) {
	s.cfg = cfg
}

// Init initializes the HTTP service
func (s *Service) Init() error {
	if s.cfg == nil {
		return nil
	}

	s.SetState(zService.ServiceStateInit)
	zLog.Info("Initializing Echo HTTP service...", zap.String("listen_address", s.cfg.HTTP.ListenAddress))

	// Register routes
	s.registerRoutes()

	return nil
}

// Start starts the HTTP service
func (s *Service) Start() error {
	if s.cfg == nil {
		zLog.Info("HTTP service config not set, skipping start")
		return nil
	}

	s.SetState(zService.ServiceStateRunning)
	zLog.Info("Starting Echo HTTP service...", zap.String("listen_address", s.cfg.HTTP.ListenAddress))

	// Start server in a goroutine
	go func() {
		if err := s.echo.Start(s.cfg.HTTP.ListenAddress); err != nil && err != http.ErrServerClosed {
			zLog.Error("Failed to start HTTP service", zap.Error(err))
		}
	}()

	s.isRunning = true
	return nil
}

// Stop stops the HTTP service
func (s *Service) Stop() error {
	if !s.isRunning {
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

	s.isRunning = false
	s.SetState(zService.ServiceStateStopped)
	zLog.Info("Echo HTTP service stopped")
	return nil
}

// Serve implements the Service interface
func (s *Service) Serve() {
	if s.cfg == nil {
		return
	}

	// Start the HTTP server
	if err := s.Start(); err != nil {
		zLog.Error("Failed to serve HTTP service", zap.Error(err))
		return
	}

	// Wait for server to stop
	select {}
}

// registerRoutes registers all API routes
func (s *Service) registerRoutes() {
	// Health check
	s.echo.GET("/health", s.handleHealthCheck)

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
		server.POST("/register", handler.HandleServerRegister)
		server.POST("/heartbeat", handler.HandleServerHeartbeat)
	}
}

// handleHealthCheck handles health check requests
func (s *Service) handleHealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "GlobalServer",
		"time":    time.Now().Format(time.RFC3339),
	})
}
