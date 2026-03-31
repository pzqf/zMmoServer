package service

import (
	"time"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zCommon/net/protolayer"
	"github.com/pzqf/zCommon/net/router"
	"github.com/pzqf/zCommon/util"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zEngine/zService"
	"go.uber.org/zap"
)

// TcpServiceConfig TCP服务配置
type TcpServiceConfig struct {
	ListenAddress     string
	ChanSize          int
	MaxClientCount    int
	HeartbeatDuration int
	Protocol          string
	DDoS              zNet.DDoSConfig
}

type TcpService struct {
	zService.BaseService
	netServer     *zNet.TcpServer
	netConfig     *zNet.TcpConfig
	packetRouter  *router.PacketRouter
	metrics       *metrics.NetworkMetrics
	protocol      protolayer.Protocol
	serviceConfig *TcpServiceConfig
}

// NewTcpService 创建TCP服务
// serviceConfig: 服务配置，包含监听地址、DDoS攻击配置
func NewTcpService(router *router.PacketRouter, serviceConfig *TcpServiceConfig) *TcpService {
	ts := &TcpService{
		BaseService:   *zService.NewBaseService(id.ServiceIdTcpServer),
		packetRouter:  router,
		metrics:       metrics.NewNetworkMetrics(),
		serviceConfig: serviceConfig,
	}
	return ts
}

func (ts *TcpService) Init() error {
	ts.SetState(zService.ServiceStateInit)

	ts.netConfig = &zNet.TcpConfig{
		ListenAddress:     ts.serviceConfig.ListenAddress,
		ChanSize:          ts.serviceConfig.ChanSize,
		MaxClientCount:    ts.serviceConfig.MaxClientCount,
		HeartbeatDuration: ts.serviceConfig.HeartbeatDuration,
	}
	zLog.Info("Initializing TCP service...", zap.String("listen_address", ts.netConfig.ListenAddress))

	// 根据配置创建协议实例
	protocolName := ts.serviceConfig.Protocol
	if protocolName == "" {
		protocolName = "protobuf" // 默认使用protobuf
	}

	var err error
	ts.protocol, err = protolayer.NewProtocolByName(protocolName)
	if err != nil {
		zLog.Warn("Failed to create protocol, using default protobuf", zap.Error(err))
		ts.protocol = protolayer.NewProtobufProtocol()
	}
	zLog.Info("Protocol initialized", zap.String("protocol", protocolName))

	// 配置防DDoS攻击参数
	ddosConfig := &ts.serviceConfig.DDoS

	// 使用标准日志接口
	logger := zLog.GetStandardLogger()
	ts.netServer = zNet.NewTcpServer(ts.netConfig, zNet.WithLogger(logger), zNet.WithDDoSConfig(ddosConfig))
	ts.netServer.RegisterDispatcher(ts.dispatchPacket)

	// 设置网络指标监控实例到protocol
	protolayer.SetNetworkMetrics(ts.metrics)

	return nil
}

func (ts *TcpService) Close() error {
	ts.SetState(zService.ServiceStateStopping)
	zLog.Info("Closing TCP service...")
	ts.netServer.Close()
	ts.SetState(zService.ServiceStateStopped)
	return nil
}

func (ts *TcpService) Serve() {
	ts.SetState(zService.ServiceStateRunning)
	zLog.Info("Starting TCP service...")

	// 启动定期打印网络指标的协程
	go ts.startMetricsPrinter()

	if err := ts.netServer.Start(); err != nil {
		zLog.Error("Failed to start TCP service", zap.Error(err))
		ts.SetState(zService.ServiceStateStopped)
		return
	}
}

// startMetricsPrinter 启动定期打印网络指标的协程
func (ts *TcpService) startMetricsPrinter() {
	defer util.Recover(func(recover interface{}, stack string) {
		zLog.Error("Metrics printer panicked", zap.Any("panic", recover))
	})
}

func (ts *TcpService) dispatchPacket(session zNet.Session, packet *zNet.NetPacket) error {
	// 记录接收的数据包大小
	ts.metrics.RecordBytesReceived(len(packet.Data) + zNet.NetPacketHeadSize)

	// 直接处理数据包，保证顺序
	defer util.Recover(func(recover interface{}, stack string) {
		zLog.Error("Packet processing panicked", zap.Any("panic", recover))
	})

	ts.processPacket(session, packet)

	return nil
}

// processPacket 处理数据包
func (ts *TcpService) processPacket(session zNet.Session, packet *zNet.NetPacket) error {
	// 记录开始处理时间
	startTime := time.Now()

	// 路由数据包到相应的处理程
	err := ts.packetRouter.Route(session, packet)

	// 记录处理延迟
	latency := time.Since(startTime)
	ts.metrics.RecordLatency(latency)

	if err != nil {
		zLog.Error("Failed to route packet", zap.Error(err))
		ts.metrics.IncDecodingErrors()
	}

	return err
}
