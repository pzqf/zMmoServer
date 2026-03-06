package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/pzqf/zMmoShared/protocol"
)

// HTTP API 相关结构体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type AuthResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Token   string `json:"token"`
	UserID  int64  `json:"user_id"`
}

type ServerInfo struct {
	ID        int32  `json:"server_id"`
	Name      string `json:"server_name"`
	Address   string `json:"address"`
	Port      int32  `json:"port"`
	Status    int32  `json:"status"`
	Online    int32  `json:"online_count"`
	MaxOnline int32  `json:"max_online_count"`
}

// GetAddr 返回完整的地址（地址:端口）
func (s *ServerInfo) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

type ServerListResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Servers []ServerInfo `json:"servers"`
}

type Client struct {
	conn             net.Conn
	globalServerAddr string
	gatewayAddr      string
	token            string
	userID           int64
	selectedServer   *ServerInfo
	stopChan         chan struct{}
}

func NewClient(globalServerAddr string) *Client {
	return &Client{
		globalServerAddr: globalServerAddr,
		stopChan:         make(chan struct{}),
	}
}

func (c *Client) Connect() error {
	conn, err := net.DialTimeout("tcp", c.gatewayAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to gateway: %v", err)
	}
	c.conn = conn
	go c.readLoop()
	return nil
}

func (c *Client) Disconnect() {
	close(c.stopChan)
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) readLoop() {
	buffer := make([]byte, 4096)
	for {
		select {
		case <-c.stopChan:
			return
		default:
			c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, err := c.conn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if err != io.EOF {
					fmt.Printf("Read error: %v\n", err)
				}
				return
			}
			if n > 0 {
				c.handleMessage(buffer[:n])
			}
		}
	}
}

func (c *Client) handleMessage(data []byte) {
	if len(data) < 8 {
		fmt.Println("Invalid message format")
		return
	}

	// 解析长度
	length := binary.BigEndian.Uint32(data[:4])
	if length > 1024*1024 {
		fmt.Println("Message too long")
		return
	}

	if len(data) < int(length) {
		fmt.Println("Insufficient data")
		return
	}

	// 解析消息ID
	msgID := binary.BigEndian.Uint32(data[4:8])

	// 解析消息内容
	switch msgID {
	case protocol.MsgIdPlayerLogin:
		var resp protocol.PlayerLoginResponse
		if err := resp.Unmarshal(data[8:length]); err != nil {
			fmt.Printf("Failed to unmarshal PlayerLoginResponse: %v\n", err)
			return
		}
		fmt.Printf("PlayerLoginResponse: Success=%v, PlayerID=%d, Name=%s, Level=%d, Gold=%d\n",
			resp.Success, resp.PlayerId, resp.Name, resp.Level, resp.Gold)
	case protocol.MsgIdPlayerCreate:
		var resp protocol.PlayerCreateResponse
		if err := resp.Unmarshal(data[8:length]); err != nil {
			fmt.Printf("Failed to unmarshal PlayerCreateResponse: %v\n", err)
			return
		}
		fmt.Printf("PlayerCreateResponse: Success=%v\n", resp.Success)
		if resp.Success && resp.Player != nil {
			fmt.Printf("Player: ID=%d, Name=%s, Level=%d, Sex=%d, Age=%d\n",
				resp.Player.PlayerId, resp.Player.Name, resp.Player.Level, resp.Player.Sex, resp.Player.Age)
		}
	case protocol.MsgIdActivityList:
		var resp protocol.ActivityListResponse
		if err := resp.Unmarshal(data[8:length]); err != nil {
			fmt.Printf("Failed to unmarshal ActivityListResponse: %v\n", err)
			return
		}
		fmt.Printf("ActivityListResponse: Success=%v, Activities=%d\n", resp.Success, len(resp.Activities))
	case protocol.MsgIdShopItemList:
		var resp protocol.ShopItemListResponse
		if err := resp.Unmarshal(data[8:length]); err != nil {
			fmt.Printf("Failed to unmarshal ShopItemListResponse: %v\n", err)
			return
		}
		fmt.Printf("ShopItemListResponse: Success=%v, Items=%d\n", resp.Success, len(resp.Items))
	case protocol.MsgIdDungeonList:
		var resp protocol.DungeonListResponse
		if err := resp.Unmarshal(data[8:length]); err != nil {
			fmt.Printf("Failed to unmarshal DungeonListResponse: %v\n", err)
			return
		}
		fmt.Printf("DungeonListResponse: Success=%v, Dungeons=%d\n", resp.Success, len(resp.Dungeons))
	case protocol.MsgIdBuffList:
		var resp protocol.BuffListResponse
		if err := resp.Unmarshal(data[8:length]); err != nil {
			fmt.Printf("Failed to unmarshal BuffListResponse: %v\n", err)
			return
		}
		fmt.Printf("BuffListResponse: Success=%v, Buffs=%d\n", resp.Success, len(resp.Buffs))
	case protocol.MsgIdDropList:
		var resp protocol.DropListResponse
		if err := resp.Unmarshal(data[8:length]); err != nil {
			fmt.Printf("Failed to unmarshal DropListResponse: %v\n", err)
			return
		}
		fmt.Printf("DropListResponse: Success=%v, Drops=%d\n", resp.Success, len(resp.Drops))
	default:
		fmt.Printf("Received message: ID=%d, Length=%d\n", msgID, length)
	}
}

func (c *Client) Send(msgID uint32, data []byte) error {
	// 构建消息：长度 + 消息ID + 数据
	length := uint32(8 + len(data))
	buffer := make([]byte, length)
	binary.BigEndian.PutUint32(buffer[:4], length)
	binary.BigEndian.PutUint32(buffer[4:8], msgID)
	copy(buffer[8:], data)

	_, err := c.conn.Write(buffer)
	return err
}

func (c *Client) SendHeartbeat() error {
	// 心跳消息
	return c.Send(1, nil)
}

func (c *Client) SendTokenVerify(token string) error {
	// 令牌验证消息
	data := []byte(token)
	return c.Send(2, data)
}

func (c *Client) SendEnterGame(serverID int32) error {
	// 进入游戏消息
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(serverID))
	return c.Send(6, data)
}

func (c *Client) SendPlayerLogin(playerID int64) error {
	// 玩家登录请求
	req := &protocol.PlayerLoginRequest{
		PlayerId: playerID,
	}
	data, err := req.Marshal()
	if err != nil {
		return err
	}
	return c.Send(protocol.MsgIdPlayerLogin, data)
}

func (c *Client) SendPlayerCreate(name string, sex, age int32) error {
	// 角色创建请求
	req := &protocol.PlayerCreateRequest{
		Name: name,
		Sex:  sex,
		Age:  age,
	}
	data, err := req.Marshal()
	if err != nil {
		return err
	}
	return c.Send(protocol.MsgIdPlayerCreate, data)
}

func (c *Client) SendPlayerLogout() error {
	// 玩家登出请求
	return c.Send(protocol.MsgIdPlayerLogout, nil)
}

func (c *Client) SendActivityList() error {
	// 活动列表请求
	return c.Send(protocol.MsgIdActivityList, nil)
}

func (c *Client) SendShopItemList(categoryID int32) error {
	// 商品列表请求
	req := &protocol.ShopItemListRequest{
		CategoryId: categoryID,
	}
	data, err := req.Marshal()
	if err != nil {
		return err
	}
	return c.Send(protocol.MsgIdShopItemList, data)
}

func (c *Client) SendDungeonList() error {
	// 副本列表请求
	return c.Send(protocol.MsgIdDungeonList, nil)
}

func (c *Client) SendBuffList() error {
	// Buff列表请求
	return c.Send(protocol.MsgIdBuffList, nil)
}

func (c *Client) SendDropList(x, y, z, radius float32) error {
	// 掉落列表请求
	req := &protocol.DropListRequest{
		X:      x,
		Y:      y,
		Z:      z,
		Radius: radius,
	}
	data, err := req.Marshal()
	if err != nil {
		return err
	}
	return c.Send(protocol.MsgIdDropList, data)
}

// HTTP 方法
func (c *Client) Login(username, password string) (*AuthResponse, error) {
	req := LoginRequest{Username: username, Password: password}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(fmt.Sprintf("http://%s/api/v1/account/login", c.globalServerAddr), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, err
	}

	if authResp.Code == 0 {
		c.token = authResp.Token
		c.userID = authResp.UserID
	}

	return &authResp, nil
}

func (c *Client) Register(username, password, email string) (*AuthResponse, error) {
	req := RegisterRequest{Username: username, Password: password, Email: email}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(fmt.Sprintf("http://%s/api/v1/account/create", c.globalServerAddr), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, err
	}

	if authResp.Code == 0 {
		c.token = authResp.Token
		c.userID = authResp.UserID
	}

	return &authResp, nil
}

func (c *Client) GetServerList() (*ServerListResponse, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/server/list", c.globalServerAddr))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var serverListResp ServerListResponse
	if err := json.Unmarshal(body, &serverListResp); err != nil {
		return nil, err
	}

	return &serverListResp, nil
}

func (c *Client) SelectServer(serverID int32, serverList []ServerInfo) bool {
	for _, server := range serverList {
		if server.ID == serverID {
			c.selectedServer = &server
			c.gatewayAddr = server.GetAddr()
			return true
		}
	}
	return false
}

func main() {
	globalServer := flag.String("global", "127.0.0.1:8082", "Global server address")
	flag.Parse()

	client := NewClient(*globalServer)

	// 1. 账号登录/注册
	fmt.Println("=== 账号登录/注册 ===")
	fmt.Println("1. 登录")
	fmt.Println("2. 注册")
	fmt.Print("请选择操作: ")

	scanner := bufio.NewScanner(os.Stdin)
	var choice string
	if scanner.Scan() {
		choice = scanner.Text()
	}

	switch choice {
	case "1":
		fmt.Print("用户名: ")
		var username string
		if scanner.Scan() {
			username = scanner.Text()
		}
		fmt.Print("密码: ")
		var password string
		if scanner.Scan() {
			password = scanner.Text()
		}

		resp, err := client.Login(username, password)
		if err != nil {
			fmt.Printf("登录失败: %v\n", err)
			os.Exit(1)
		}
		if resp.Code != 0 {
			fmt.Printf("登录失败: %s\n", resp.Message)
			os.Exit(1)
		}
		fmt.Printf("登录成功! 用户ID: %d\n", resp.UserID)

	case "2":
		fmt.Print("用户名: ")
		var username string
		if scanner.Scan() {
			username = scanner.Text()
		}
		fmt.Print("密码: ")
		var password string
		if scanner.Scan() {
			password = scanner.Text()
		}
		fmt.Print("邮箱: ")
		var email string
		if scanner.Scan() {
			email = scanner.Text()
		}

		resp, err := client.Register(username, password, email)
		if err != nil {
			fmt.Printf("注册失败: %v\n", err)
			os.Exit(1)
		}
		if resp.Code != 0 {
			fmt.Printf("注册失败: %s\n", resp.Message)
			os.Exit(1)
		}
		fmt.Printf("注册成功! 用户ID: %d\n", resp.UserID)

	default:
		fmt.Println("无效选择")
		os.Exit(1)
	}

	// 2. 获取服务器列表
	fmt.Println("\n=== 服务器列表 ===")
	serverListResp, err := client.GetServerList()
	if err != nil {
		fmt.Printf("获取服务器列表失败: %v\n", err)
		os.Exit(1)
	}
	if serverListResp.Code != 0 {
		fmt.Printf("获取服务器列表失败: %s\n", serverListResp.Message)
		os.Exit(1)
	}

	for i, server := range serverListResp.Servers {
		fmt.Printf("%d. %s (状态: %d, 在线: %d/%d)\n", i+1, server.Name, server.Status, server.Online, server.MaxOnline)
	}

	// 3. 选择服务器
	fmt.Print("\n请选择服务器: ")
	var serverIndex int
	if scanner.Scan() {
		fmt.Sscanf(scanner.Text(), "%d", &serverIndex)
	}

	if serverIndex < 1 || serverIndex > len(serverListResp.Servers) {
		fmt.Println("无效服务器选择")
		os.Exit(1)
	}

	targetServer := serverListResp.Servers[serverIndex-1]
	if !client.SelectServer(targetServer.ID, serverListResp.Servers) {
		fmt.Println("服务器选择失败")
		os.Exit(1)
	}

	fmt.Printf("已选择服务器: %s (%s)\n", targetServer.Name, targetServer.GetAddr())

	// 4. 连接到 Gateway 服务器
	fmt.Println("\n=== 连接 Gateway 服务器 ===")
	if err := client.Connect(); err != nil {
		fmt.Printf("连接失败: %v\n", err)
		os.Exit(1)
	}
	defer client.Disconnect()

	fmt.Println("连接成功!")

	// 5. 发送令牌验证
	if err := client.SendTokenVerify(client.token); err != nil {
		fmt.Printf("发送令牌失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("令牌验证成功!")

	// 6. 进入游戏
	fmt.Println("\n=== 进入游戏 ===")
	if err := client.SendEnterGame(targetServer.ID); err != nil {
		fmt.Printf("进入游戏失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("进入游戏成功! 开始游戏...")

	// 7. 角色创建/登录
	fmt.Println("\n=== 角色管理 ===")
	fmt.Println("1. 创建角色")
	fmt.Println("2. 登录角色")
	fmt.Print("请选择操作: ")
	var roleChoice string
	if scanner.Scan() {
		roleChoice = scanner.Text()
	}

	switch roleChoice {
	case "1":
		fmt.Print("角色名称: ")
		var name string
		if scanner.Scan() {
			name = scanner.Text()
		}
		fmt.Print("性别 (1-男, 2-女): ")
		var sex int32
		if scanner.Scan() {
			fmt.Sscanf(scanner.Text(), "%d", &sex)
		}
		fmt.Print("年龄: ")
		var age int32
		if scanner.Scan() {
			fmt.Sscanf(scanner.Text(), "%d", &age)
		}

		if err := client.SendPlayerCreate(name, sex, age); err != nil {
			fmt.Printf("创建角色失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("角色创建请求已发送，等待响应...")

	case "2":
		fmt.Print("角色ID: ")
		var playerID int64
		if scanner.Scan() {
			fmt.Sscanf(scanner.Text(), "%d", &playerID)
		}

		if err := client.SendPlayerLogin(playerID); err != nil {
			fmt.Printf("登录角色失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("角色登录请求已发送，等待响应...")

	default:
		fmt.Println("无效选择")
	}

	// 进入游戏后，保持连接并提供命令交互
	fmt.Println("\n游戏中... 输入命令:")
	fmt.Println("- 'quit' 退出")
	fmt.Println("- 'logout' 登出角色")
	fmt.Println("- 'activity' 查看活动列表")
	fmt.Println("- 'shop' 查看商城商品")
	fmt.Println("- 'dungeon' 查看副本列表")
	fmt.Println("- 'buff' 查看Buff列表")
	fmt.Println("- 'drop' 查看掉落列表")

	for scanner.Scan() {
		line := scanner.Text()
		if line == "quit" || line == "exit" {
			break
		} else if line == "logout" {
			if err := client.SendPlayerLogout(); err != nil {
				fmt.Printf("登出失败: %v\n", err)
			} else {
				fmt.Println("登出请求已发送")
			}
		} else if line == "activity" {
			if err := client.SendActivityList(); err != nil {
				fmt.Printf("获取活动列表失败: %v\n", err)
			} else {
				fmt.Println("活动列表请求已发送")
			}
		} else if line == "shop" {
			if err := client.SendShopItemList(1); err != nil {
				fmt.Printf("获取商品列表失败: %v\n", err)
			} else {
				fmt.Println("商品列表请求已发送")
			}
		} else if line == "dungeon" {
			if err := client.SendDungeonList(); err != nil {
				fmt.Printf("获取副本列表失败: %v\n", err)
			} else {
				fmt.Println("副本列表请求已发送")
			}
		} else if line == "buff" {
			if err := client.SendBuffList(); err != nil {
				fmt.Printf("获取Buff列表失败: %v\n", err)
			} else {
				fmt.Println("Buff列表请求已发送")
			}
		} else if line == "drop" {
			if err := client.SendDropList(0, 0, 0, 100); err != nil {
				fmt.Printf("获取掉落列表失败: %v\n", err)
			} else {
				fmt.Println("掉落列表请求已发送")
			}
		} else {
			fmt.Println("未知命令")
		}
	}

	fmt.Println("游戏结束!")
}
