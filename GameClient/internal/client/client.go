package client

import (
	"fmt"
	"net"
	"strconv"
	"time"

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
	// 解析 gatewayAddr（"host:port"）——解析失败直接返回 error，不静默回退到错误默认端口。
	host, portStr, err := net.SplitHostPort(c.gatewayAddr)
	if err != nil {
		return fmt.Errorf("invalid gateway address %q: %v", c.gatewayAddr, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid gateway port %q: %v", portStr, err)
	}

	// 创建TcpClientConfig
	config := &zNet.TcpClientConfig{
		ServerAddr:        host,
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
			// addr 无效(0.0.0.0/空)时保留命令行传入的 gatewayAddr，否则采用服务器返回地址。
			if addr != "0.0.0.0" && addr != "" {
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

// —— 物品 / 仓库（业务层建设 2026-07-25）——
func (c *Client) SendItemList(playerID int64) error { return c.messageSender.SendItemList(playerID) }
func (c *Client) SendItemUse(playerID int64, slot, count int32) error {
	return c.messageSender.SendItemUse(playerID, slot, count)
}
func (c *Client) SendItemPickup(playerID int64, mapID int32) error {
	return c.messageSender.SendItemPickup(playerID, mapID)
}
func (c *Client) SendChat(playerID int64, channel int32, text string) error {
	return c.messageSender.SendChat(playerID, channel, text)
}
func (c *Client) SendTeamCreate(playerID int64) error { return c.messageSender.SendTeamCreate(playerID) }
func (c *Client) SendTeamJoin(playerID int64, teamID int32) error {
	return c.messageSender.SendTeamJoin(playerID, teamID)
}
func (c *Client) SendTeamLeave(playerID int64) error { return c.messageSender.SendTeamLeave(playerID) }
func (c *Client) SendTradeStart(playerID, targetID int64) error {
	return c.messageSender.SendTradeStart(playerID, targetID)
}
func (c *Client) SendTradeSetGold(playerID, gold int64) error {
	return c.messageSender.SendTradeSetGold(playerID, gold)
}
func (c *Client) SendTradeConfirm(playerID int64) error {
	return c.messageSender.SendTradeConfirm(playerID)
}

// EnableTradeAuto 开启交易自动应价（本方收到 OPEN 通知即设价+确认，事件驱动收敛到成交）。
func (c *Client) EnableTradeAuto(playerID, gold int64) {
	c.messageHandler.EnableTradeAuto(playerID, gold,
		func(g int64) { _ = c.SendTradeSetGold(playerID, g) },
		func() { _ = c.SendTradeConfirm(playerID) })
}

func (c *Client) SendMail(fromID, toID int64, title, content string, gold int64) error {
	return c.messageSender.SendMail(fromID, toID, title, content, gold)
}
func (c *Client) SendMailList(playerID int64) error { return c.messageSender.SendMailList(playerID) }
func (c *Client) SendMailClaim(playerID, mailID int64) error {
	return c.messageSender.SendMailClaim(playerID, mailID)
}

// EnableMailAutoClaim 开启邮件自动领取（收到邮件列表后逐封领取未领）。
func (c *Client) EnableMailAutoClaim(playerID int64) {
	c.messageHandler.EnableMailAutoClaim(func(mailID int64) { _ = c.SendMailClaim(playerID, mailID) })
}
func (c *Client) SendItemMove(playerID int64, from, to int32) error {
	return c.messageSender.SendItemMove(playerID, from, to)
}
func (c *Client) SendWarehouseList(playerID int64) error {
	return c.messageSender.SendWarehouseList(playerID)
}
func (c *Client) SendWarehouseStore(playerID int64, bagSlot, count int32) error {
	return c.messageSender.SendWarehouseStore(playerID, bagSlot, count)
}
func (c *Client) SendWarehouseRetrieve(playerID int64, warehouseSlot, count int32) error {
	return c.messageSender.SendWarehouseRetrieve(playerID, warehouseSlot, count)
}

// —— 技能（业务层建设 2026-07-25）——
func (c *Client) SendSkillList(playerID int64) error { return c.messageSender.SendSkillList(playerID) }
func (c *Client) SendSkillLearn(playerID int64, skillID int32) error {
	return c.messageSender.SendSkillLearn(playerID, skillID)
}
func (c *Client) SendSkillUpgrade(playerID int64, skillID int32) error {
	return c.messageSender.SendSkillUpgrade(playerID, skillID)
}
func (c *Client) SendSkillCast(playerID int64, skillID int32, targetID int64) error {
	return c.messageSender.SendSkillCast(playerID, skillID, targetID)
}

// SetGatewayAddr 设置网关地址
func (c *Client) SetGatewayAddr(addr string) {
	c.gatewayAddr = addr
}

// GetToken 获取当前token
func (c *Client) GetToken() string {
	return c.token
}

// SetPlayerID 透传当前玩家ID到 sender（消除对隐藏状态的依赖）。
// -loginPlayerID>0 复用已有角色、跳过创建时，须显式调用它，
// 否则后续 SendPlayerLogout 读 sender.playerID 仍为 0，会登出玩家 0。
func (c *Client) SetPlayerID(playerID int64) {
	if c.messageSender != nil {
		c.messageSender.SetPlayerID(playerID)
	}
}

// —— 关键屏障：发送后等对应响应且判 Result（取代盲等 sleep）——

// WaitTokenVerify 等待 token 验证响应，返回其 Result 与是否在超时内到达。
func (c *Client) WaitTokenVerify(timeout time.Duration) (int32, bool) {
	return c.messageHandler.WaitFor(uint32(protocol.SystemMsgId_MSG_SYSTEM_TOKEN_VERIFY_RESPONSE), timeout)
}

// WaitPlayerLogin 等待进入游戏响应。
func (c *Client) WaitPlayerLogin(timeout time.Duration) (int32, bool) {
	return c.messageHandler.WaitFor(uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE), timeout)
}

// WaitMapEnter 等待进入地图响应。
func (c *Client) WaitMapEnter(timeout time.Duration) (int32, bool) {
	return c.messageHandler.WaitFor(uint32(protocol.MapMsgId_MSG_MAP_ENTER_RESPONSE), timeout)
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
