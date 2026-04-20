package client

import (
	"fmt"

	"github.com/pzqf/zCommon/protocol"
	"github.com/pzqf/zEngine/zNet"
	"github.com/pzqf/zMmoServer/GameClient/internal/http"
	"github.com/pzqf/zMmoServer/GameClient/internal/msg/handler"
	"github.com/pzqf/zMmoServer/GameClient/internal/msg/sender"
)

type Client struct {
	tcpClient        *zNet.TcpClient
	messageHandler   *handler.MessageHandler
	messageSender    *sender.MessageSender
	httpClient       *http.Client
	globalServerAddr string
	gatewayAddr      string
	token            string
	selectedServer   *protocol.ServerInfo
}

func NewClient(globalServerAddr string) *Client {
	httpClient := http.NewClient(globalServerAddr)
	messageHandler := handler.NewMessageHandler()

	return &Client{
		globalServerAddr: globalServerAddr,
		httpClient:       httpClient,
		messageHandler:   messageHandler,
	}
}

func (c *Client) Connect() error {
	// 解析gatewayAddr，提取出地址和端口
	addr := c.gatewayAddr
	port := 8080
	// 简单解析，假设格式为 "address:port"
	if len(addr) > 0 {
		for i := len(addr) - 1; i >= 0; i-- {
			if addr[i] == ':' {
				portStr := addr[i+1:]
				addr = addr[:i]
				if p, err := fmt.Sscanf(portStr, "%d", &port); err == nil && p == 1 {
					break
				}
			}
		}
	}

	// 创建TcpClientConfig
	config := &zNet.TcpClientConfig{
		ServerAddr:        addr,
		ServerPort:        port,
		HeartbeatDuration: 30,
		MaxPacketDataSize: 1024 * 1024,
		AutoReconnect:     true,
		ReconnectDelay:    5,
		MaxReconnectTimes: 10,
		DisableEncryption: false,
		ChanSize:          100,
		Compression:       zNet.CompressionConfig{},
	}

	// 创建TcpClient实例
	c.tcpClient = zNet.NewTcpClient(config)

	// 注册消息处理器
	c.tcpClient.RegisterDispatcher(c.messageHandler.GetDispatcher())

	// 创建消息发送器
	c.messageSender = sender.NewMessageSender(c.tcpClient, c.token)

	// 连接服务器
	if err := c.tcpClient.Connect(); err != nil {
		return fmt.Errorf("failed to connect to gateway: %v", err)
	}

	return nil
}

func (c *Client) Disconnect() {
	if c.tcpClient != nil {
		c.tcpClient.Close()
	}
}

// HTTP 方法
func (c *Client) Login(account, password string) (*http.AuthResponse, error) {
	authResp, err := c.httpClient.Login(account, password)
	if err != nil {
		return nil, err
	}

	if authResp.Result == 0 {
		c.token = authResp.Token
		if len(authResp.Servers) > 0 {
			c.selectedServer = authResp.Servers[0]
			addr := c.selectedServer.Address
			if addr == "0.0.0.0" || addr == "" {
				if c.gatewayAddr != "" {
					// 保留命令行传入的gateway地址
				}
			} else {
				c.gatewayAddr = fmt.Sprintf("%s:%d", addr, c.selectedServer.Port)
			}
		}
	}

	return authResp, nil
}

func (c *Client) Register(account, password, email string) (*http.AuthResponse, error) {
	return c.httpClient.Register(account, password, email)
}

func (c *Client) GetServerList() (*http.ServerListResponse, error) {
	return c.httpClient.GetServerList()
}

func (c *Client) SelectServer(serverID int32, serverList []*protocol.ServerInfo) bool {
	server, addr := c.httpClient.SelectServer(serverID, serverList)
	if server != nil {
		c.selectedServer = server
		c.gatewayAddr = addr
		return true
	}
	return false
}

// 消息发送方法
func (c *Client) SendHeartbeat() error {
	return c.messageSender.SendHeartbeat()
}

func (c *Client) SendTokenVerify(token string) error {
	return c.messageSender.SendTokenVerify(token)
}

func (c *Client) SendPlayerLogin(playerID int64) error {
	return c.messageSender.SendPlayerLogin(playerID)
}

func (c *Client) SendPlayerCreate(name string, sex, age int32) error {
	return c.messageSender.SendPlayerCreate(name, sex, age)
}

func (c *Client) SendPlayerLogout() error {
	return c.messageSender.SendPlayerLogout()
}

func (c *Client) SendMapEnter(playerID int64, mapID int32) error {
	return c.messageSender.SendMapEnter(playerID, mapID)
}

func (c *Client) SendMapMove(playerID int64, mapID int32, x, y, z float32) error {
	return c.messageSender.SendMapMove(playerID, mapID, x, y, z)
}

func (c *Client) SendMapAttack(playerID int64, mapID int32, targetID int64) error {
	return c.messageSender.SendMapAttack(playerID, mapID, targetID)
}

// SetGatewayAddr 设置网关地址
func (c *Client) SetGatewayAddr(addr string) {
	c.gatewayAddr = addr
}

// GetToken 获取当前token
func (c *Client) GetToken() string {
	return c.token
}

// SetToken 设置token
func (c *Client) SetToken(token string) {
	c.token = token
	if c.messageSender != nil {
		c.messageSender.SetToken(token)
	}
}

// SelectedServer 获取选中的服务器
func (c *Client) SelectedServer() *protocol.ServerInfo {
	return c.selectedServer
}

func (c *Client) GetCreatedPlayerID() int64 {
	if c.messageHandler != nil {
		pid := c.messageHandler.WaitForPlayerID()
		if pid != 0 && c.messageSender != nil {
			c.messageSender.SetPlayerID(pid)
		}
		return pid
	}
	return 0
}
