package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/net/router"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zEngine/zService"
	"go.uber.org/zap"
)

type HTTPHandlerFunc func(w http.ResponseWriter, r *http.Request)

type RouteMap map[string]HTTPHandlerFunc

// HTTPServiceConfig HTTP服务配置
type HTTPServiceConfig struct {
	ListenAddress     string
	MaxClientCount    int
	MaxPacketDataSize int32
	Enabled           bool
	DDoS              zNet.DDoSConfig
}

type HTTPService struct {
	zService.BaseService
	server        *http.Server
	httpServer    *zNet.HttpServer
	routes        RouteMap
	mux           *http.ServeMux
	packetRouter  *router.PacketRouter
	serviceConfig *HTTPServiceConfig
	enabled       bool
}

// NewHTTPService 创建HTTP服务
// serviceConfig: 服务配置，包含监听地址、DDoS配置
func NewHTTPService(packetRouter *router.PacketRouter, serviceConfig *HTTPServiceConfig) *HTTPService {
	hs := &HTTPService{
		BaseService:   *zService.NewBaseService(id.ServiceIdHttpServer),
		routes:        make(RouteMap),
		mux:           http.NewServeMux(),
		packetRouter:  packetRouter,
		serviceConfig: serviceConfig,
		enabled:       serviceConfig != nil && serviceConfig.Enabled,
	}
	return hs
}

func (hs *HTTPService) Init() error {
	hs.SetState(zService.ServiceStateInit)

	if !hs.enabled {
		zLog.Info("HTTP service is disabled")
		return nil
	}

	zLog.Info("Initializing HTTP service...", zap.String("listen_address", hs.serviceConfig.ListenAddress))

	ddosConfig := &hs.serviceConfig.DDoS

	httpConfig := &zNet.HttpConfig{
		ListenAddress:     hs.serviceConfig.ListenAddress,
		MaxClientCount:    hs.serviceConfig.MaxClientCount,
		MaxPacketDataSize: hs.serviceConfig.MaxPacketDataSize,
	}

	hs.httpServer = zNet.NewHttpServer(httpConfig, zNet.WithDDoSConfig(ddosConfig))

	if hs.packetRouter != nil {
		hs.httpServer.RegisterDispatcher(hs.dispatchPacket)
	}

	hs.registerDefaultRoutes()

	// 注册 /proto 路径处理函数
	hs.RegisterHandler("/proto", hs.handleProtoRequest)

	return nil
}

func (hs *HTTPService) dispatchPacket(session zNet.Session, packet *zNet.NetPacket) error {
	if hs.packetRouter == nil {
		return nil
	}
	return hs.packetRouter.Route(session, packet)
}

func (hs *HTTPService) Close() error {
	if !hs.enabled {
		return nil
	}

	hs.SetState(zService.ServiceStateStopping)
	zLog.Info("Closing HTTP service...")
	if hs.httpServer != nil {
		hs.httpServer.Close()
	}
	hs.SetState(zService.ServiceStateStopped)
	return nil
}

func (hs *HTTPService) Serve() {
	if !hs.enabled {
		zLog.Info("HTTP service is disabled, skipping start")
		return
	}

	hs.SetState(zService.ServiceStateRunning)
	zLog.Info("Starting HTTP service...")
	if hs.httpServer != nil {
		if err := hs.httpServer.Start(); err != nil {
			zLog.Error("Failed to start HTTP service", zap.Error(err))
			hs.SetState(zService.ServiceStateStopped)
			return
		}
	}
}

func (hs *HTTPService) RegisterHandler(path string, handler HTTPHandlerFunc) {
	hs.routes[path] = handler
	hs.mux.HandleFunc(path, handler)

	if hs.httpServer != nil {
		hs.httpServer.HandleFunc(path, handler)
	}

	zLog.Debug("Registered HTTP handler", zap.String("path", path))
}

func (hs *HTTPService) UnregisterHandler(path string) {
	delete(hs.routes, path)
	hs.recreateMux()
	zLog.Debug("Unregistered HTTP handler", zap.String("path", path))
}

func (hs *HTTPService) recreateMux() {
	newMux := http.NewServeMux()
	for path, handler := range hs.routes {
		newMux.HandleFunc(path, handler)
	}
	hs.mux = newMux
}

func (hs *HTTPService) registerDefaultRoutes() {
	hs.RegisterHandler("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	hs.RegisterHandler("/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Server is running")
	})

	hs.RegisterHandler("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// 使用默认的prometheus注册表
		promhttp.Handler().ServeHTTP(w, r)
	})
}

func (hs *HTTPService) handleProtoRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 解析 JSON 请求
	type Request struct {
		MsgID int32  `json:"msg_id"`
		Data  string `json:"data"`
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	// 创建 HTTP 会话
	session := zNet.NewHttpSession(w, 0, r.RemoteAddr)

	// 处理请求
	if hs.packetRouter != nil {
		// 将请求数据转换为 []byte
		data := []byte(req.Data)

		// 创建 NetPacket
		packet := &zNet.NetPacket{
			ProtoId:  req.MsgID,
			Data:     data,
			DataSize: int32(len(data)),
		}

		// 路由数据包
		err := hs.packetRouter.Route(session, packet)
		if err != nil {
			zLog.Error("Failed to route packet", zap.Error(err), zap.Int32("msgId", req.MsgID))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}
