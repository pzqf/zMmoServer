package main

import (
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pzqf/zMmoShared/protocol"
	"github.com/pzqf/zUtil/zCrypto"
)

// HTTP API 相关结构体
type LoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type AuthResponse struct {
	Result    int32                  `json:"result"`
	ErrorMsg  string                 `json:"error_msg"`
	Token     string                 `json:"token"`
	AccountId int64                  `json:"account_id,omitempty"`
	Servers   []*protocol.ServerInfo `json:"servers,omitempty"`
}

// getServerAddr 返回完整的地址（地址:端口）
func getServerAddr(server *protocol.ServerInfo) string {
	return fmt.Sprintf("%s:%d", server.Address, server.Port)
}

type ServerListResponse struct {
	Result   int32                  `json:"result"`
	ErrorMsg string                 `json:"error_msg"`
	Servers  []*protocol.ServerInfo `json:"servers"`
}

type Client struct {
	conn             net.Conn
	globalServerAddr string
	gatewayAddr      string
	token            string
	userID           int64
	selectedServer   *protocol.ServerInfo
	stopChan         chan struct{}
	aesKey           []byte
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

	// 执行DH密钥交换
	if err := c.performKeyExchange(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to perform key exchange: %v", err)
	}

	go c.readLoop()
	return nil
}

// performKeyExchange 执行DH密钥交换
func (c *Client) performKeyExchange() error {
	// 步骤1：创建DH密钥交换实例
	curve := elliptic.P256()
	privateKey, publicKeyX, publicKeyY, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate DH key pair: %v", err)
	}

	// 步骤2：发送自己的公钥（64字节）
	xBytes := publicKeyX.Bytes()
	yBytes := publicKeyY.Bytes()

	xBytes32 := make([]byte, 32)
	yBytes32 := make([]byte, 32)

	copy(xBytes32[32-len(xBytes):], xBytes)
	copy(yBytes32[32-len(yBytes):], yBytes)

	publicKey := append(xBytes32, yBytes32...)
	if _, err := c.conn.Write(publicKey); err != nil {
		return fmt.Errorf("failed to send public key: %v", err)
	}

	// 步骤3：接收对方的公钥（64字节）
	peerPublicKey := make([]byte, 64)
	if _, err := io.ReadFull(c.conn, peerPublicKey); err != nil {
		return fmt.Errorf("failed to receive peer public key: %v", err)
	}

	// 步骤4：计算共享密钥
	peerX := new(big.Int).SetBytes(peerPublicKey[:32])
	peerY := new(big.Int).SetBytes(peerPublicKey[32:])

	if !curve.IsOnCurve(peerX, peerY) {
		return fmt.Errorf("invalid peer public key: not on curve")
	}

	x, _ := curve.ScalarMult(peerX, peerY, privateKey)
	if x == nil {
		return fmt.Errorf("failed to compute shared secret")
	}

	hash := sha256.Sum256(x.Bytes())
	c.aesKey = hash[:16]

	fmt.Printf("DH key exchange completed successfully, AES key length: %d\n", len(c.aesKey))
	return nil
}

func (c *Client) Disconnect() {
	close(c.stopChan)
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) readLoop() {
	// 先读取16字节的NetPacket头部
	headerBuffer := make([]byte, 16)
	for {
		select {
		case <-c.stopChan:
			return
		default:
			c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))

			// 读取NetPacket头部
			_, err := io.ReadFull(c.conn, headerBuffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if err != io.EOF {
					fmt.Printf("Read header error: %v\n", err)
				}
				return
			}

			// 解析NetPacket头部
			protoId := binary.LittleEndian.Uint32(headerBuffer[0:4])
			_ = binary.LittleEndian.Uint32(headerBuffer[4:8]) // version
			dataSize := binary.LittleEndian.Uint32(headerBuffer[8:12])
			_ = binary.LittleEndian.Uint32(headerBuffer[12:16]) // isCompressed

			// 读取数据体
			if dataSize > 0 {
				dataBuffer := make([]byte, dataSize)
				_, err := io.ReadFull(c.conn, dataBuffer)
				if err != nil {
					fmt.Printf("Read data error: %v\n", err)
					return
				}

				// 解密数据
				var decryptedData []byte
				if c.aesKey != nil {
					decryptedData, err = zCrypto.AESDecrypt(dataBuffer, c.aesKey, nil, zCrypto.AESModeGCM)
					if err != nil {
						fmt.Printf("AES-GCM decrypt error: %v\n", err)
						return
					}
				} else {
					decryptedData = dataBuffer
				}

				// 处理消息
				c.handleMessage(protoId, decryptedData)
			} else {
				// 处理没有数据的消息（如心跳）
				c.handleMessage(protoId, nil)
			}
		}
	}
}

func (c *Client) handleMessage(protoId uint32, data []byte) {
	// 解析消息内容
	switch protoId {
	case uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME_RESPONSE):
		var resp protocol.PlayerLoginResponse
		if err := proto.Unmarshal(data, &resp); err != nil {
			fmt.Printf("Failed to unmarshal PlayerLoginResponse: %v\n", err)
			return
		}
		fmt.Printf("PlayerLoginResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
		if resp.Result == 0 && resp.PlayerInfo != nil {
			fmt.Printf("Player: ID=%d, Name=%s, Level=%d, Gold=%d\n",
				resp.PlayerInfo.PlayerId, resp.PlayerInfo.Name, resp.PlayerInfo.Level, resp.PlayerInfo.Gold)
		}
	case uint32(protocol.PlayerMsgId_MSG_PLAYER_CREATE_RESPONSE):
		var resp protocol.PlayerCreateResponse
		if err := proto.Unmarshal(data, &resp); err != nil {
			fmt.Printf("Failed to unmarshal PlayerCreateResponse: %v\n", err)
			return
		}
		fmt.Printf("PlayerCreateResponse: Result=%d, ErrorMsg=%s\n", resp.Result, resp.ErrorMsg)
		if resp.Result == 0 && resp.PlayerInfo != nil {
			fmt.Printf("Player: ID=%d, Name=%s, Level=%d, Sex=%d, Age=%d\n",
				resp.PlayerInfo.PlayerId, resp.PlayerInfo.Name, resp.PlayerInfo.Level, 0, 0)
		}
	case uint32(protocol.ActivityMsgId_MSG_ACTIVITY_GET_LIST_RESPONSE):
		var resp protocol.ActivityListResponse
		if err := proto.Unmarshal(data, &resp); err != nil {
			fmt.Printf("Failed to unmarshal ActivityListResponse: %v\n", err)
			return
		}
		fmt.Printf("ActivityListResponse: Result=%d, ErrorMsg=%s, Activities=%d\n", resp.Result, resp.ErrorMsg, len(resp.Activities))
	case uint32(protocol.ShopMsgId_MSG_SHOP_GET_ITEMS_RESPONSE):
		var resp protocol.ShopItemListResponse
		if err := proto.Unmarshal(data, &resp); err != nil {
			fmt.Printf("Failed to unmarshal ShopItemListResponse: %v\n", err)
			return
		}
		fmt.Printf("ShopItemListResponse: Result=%d, ErrorMsg=%s, Items=%d\n", resp.Result, resp.ErrorMsg, len(resp.Items))
	case uint32(protocol.DungeonMsgId_MSG_DUNGEON_GET_LIST_RESPONSE):
		var resp protocol.DungeonListResponse
		if err := proto.Unmarshal(data, &resp); err != nil {
			fmt.Printf("Failed to unmarshal DungeonListResponse: %v\n", err)
			return
		}
		fmt.Printf("DungeonListResponse: Result=%d, ErrorMsg=%s, Dungeons=%d\n", resp.Result, resp.ErrorMsg, len(resp.Dungeons))
	case uint32(protocol.BagMsgId_MSG_BAG_GET_ITEMS_RESPONSE):
		var resp protocol.BuffListResponse
		if err := proto.Unmarshal(data, &resp); err != nil {
			fmt.Printf("Failed to unmarshal BuffListResponse: %v\n", err)
			return
		}
		fmt.Printf("BuffListResponse: Result=%d, ErrorMsg=%s, Buffs=%d\n", resp.Result, resp.ErrorMsg, len(resp.Buffs))
	case uint32(protocol.TradeMsgId_MSG_TRADE_REQUEST):
		var resp protocol.DropListResponse
		if err := proto.Unmarshal(data, &resp); err != nil {
			fmt.Printf("Failed to unmarshal DropListResponse: %v\n", err)
			return
		}
		fmt.Printf("DropListResponse: Result=%d, ErrorMsg=%s, Drops=%d\n", resp.Result, resp.ErrorMsg, len(resp.Drops))
	default:
		fmt.Printf("Received message: ProtoId=%d, DataSize=%d\n", protoId, len(data))
	}
}

func (c *Client) Send(msgID uint32, data []byte) error {
	// 构建NetPacket：ProtoId(4) + Version(4) + DataSize(4) + IsCompressed(4) + Data
	version := int32(1)
	isCompressed := int32(0)

	// 加密数据
	var encryptedData []byte
	if c.aesKey != nil && len(data) > 0 {
		var err error
		encryptedData, err = zCrypto.AESEncrypt(data, c.aesKey, nil, zCrypto.AESModeGCM)
		if err != nil {
			return fmt.Errorf("failed to encrypt data: %v", err)
		}
	} else {
		encryptedData = data
	}

	dataSize := int32(len(encryptedData))

	// 计算总长度：16字节头部 + 数据
	totalSize := 16 + dataSize
	buffer := make([]byte, totalSize)

	// 使用大端序写入头部（与zNet库一致）
	binary.BigEndian.PutUint32(buffer[0:4], uint32(msgID))
	binary.BigEndian.PutUint32(buffer[4:8], uint32(version))
	binary.BigEndian.PutUint32(buffer[8:12], uint32(dataSize))
	binary.BigEndian.PutUint32(buffer[12:16], uint32(isCompressed))

	// 写入数据
	if dataSize > 0 {
		copy(buffer[16:], encryptedData)
	}

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
	// 进入游戏消息 - 这里暂时不发送，因为需要先创建角色
	// 后续会在创建角色后调用SendPlayerLogin
	return nil
}

func (c *Client) SendPlayerLogin(playerID int64) error {
	// 玩家登录请求
	req := &protocol.PlayerLoginRequest{
		PlayerId: playerID,
		Token:    c.token,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return c.Send(uint32(protocol.PlayerMsgId_MSG_PLAYER_ENTER_GAME), data)
}

func (c *Client) SendPlayerCreate(name string, sex, age int32) error {
	// 角色创建请求
	req := &protocol.PlayerCreateRequest{
		Name:       name,
		Sex:        sex,
		Age:        age,
		Profession: 1, // 默认职业
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return c.Send(uint32(protocol.PlayerMsgId_MSG_PLAYER_CREATE), data)
}

func (c *Client) SendPlayerLogout() error {
	// 玩家登出请求
	req := &protocol.PlayerLogoutRequest{
		PlayerId: 0, // 暂时设为0
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return c.Send(uint32(protocol.PlayerMsgId_MSG_PLAYER_LEAVE_GAME), data)
}

func (c *Client) SendActivityList() error {
	// 活动列表请求
	return c.Send(uint32(protocol.ActivityMsgId_MSG_ACTIVITY_GET_LIST), nil)
}

func (c *Client) SendShopItemList(categoryID int32) error {
	// 商品列表请求
	return c.Send(uint32(protocol.ShopMsgId_MSG_SHOP_GET_ITEMS), nil)
}

func (c *Client) SendMapEnter(playerID int64, mapID int32) error {
	// 进入地图请求
	req := &protocol.ClientMapEnterRequest{
		PlayerId: playerID,
		MapId:    mapID,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return c.Send(uint32(protocol.MapMsgId_MSG_MAP_ENTER), data)
}

func (c *Client) SendMapMove(playerID int64, mapID int32, x, y, z float32) error {
	// 移动请求
	req := &protocol.ClientMapMoveRequest{
		PlayerId: playerID,
		MapId:    mapID,
		Pos: &protocol.Position{
			X: x,
			Y: y,
			Z: z,
		},
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return c.Send(uint32(protocol.MapMsgId_MSG_MAP_MOVE), data)
}

func (c *Client) SendMapAttack(playerID int64, mapID int32, targetID int64) error {
	// 攻击请求
	req := &protocol.ClientMapAttackRequest{
		PlayerId: playerID,
		MapId:    mapID,
		TargetId: targetID,
	}
	data, err := proto.Marshal(req)
	if err != nil {
		return err
	}
	return c.Send(uint32(protocol.MapMsgId_MSG_MAP_ATTACK), data)
}

func (c *Client) SendDungeonList() error {
	// 副本列表请求
	return c.Send(uint32(protocol.DungeonMsgId_MSG_DUNGEON_GET_LIST), nil)
}

func (c *Client) SendBuffList() error {
	// Buff列表请求
	return c.Send(uint32(protocol.BagMsgId_MSG_BAG_GET_ITEMS), nil)
}

func (c *Client) SendDropList(x, y, z, radius float32) error {
	// 掉落列表请求
	return c.Send(uint32(protocol.TradeMsgId_MSG_TRADE_REQUEST), nil)
}

// HTTP 方法
func (c *Client) Login(account, password string) (*AuthResponse, error) {
	req := LoginRequest{Account: account, Password: password}
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

	if authResp.Result == 0 {
		c.token = authResp.Token
		// 从服务器列表中获取第一个服务器的地址作为默认网关地址
		if len(authResp.Servers) > 0 {
			c.selectedServer = authResp.Servers[0]
			c.gatewayAddr = getServerAddr(c.selectedServer)
		}
	}

	return &authResp, nil
}

func (c *Client) Register(account, password, email string) (*AuthResponse, error) {
	req := RegisterRequest{Account: account, Password: password, Email: email}
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

func (c *Client) SelectServer(serverID int32, serverList []*protocol.ServerInfo) bool {
	for _, server := range serverList {
		if server.ServerId == serverID {
			c.selectedServer = server
			c.gatewayAddr = getServerAddr(server)
			return true
		}
	}
	return false
}

func main() {
	fmt.Println("=== 简化测试流程 ===")

	// 直接创建客户端并连接到GatewayServer
	client := NewClient("")
	client.gatewayAddr = "127.0.0.1:10001" // GatewayServer地址

	// 1. 连接到 Gateway 服务器
	fmt.Println("1. 连接 Gateway 服务器...")
	if err := client.Connect(); err != nil {
		fmt.Printf("连接失败: %v\n", err)
		os.Exit(1)
	}
	defer client.Disconnect()

	fmt.Println("连接成功!")

	// 2. 生成并发送token验证
	fmt.Println("\n2. 发送token验证...")
	secretKey := "zMmoServerSecretKey" // 与GatewayServer配置中的JWTSecret一致
	token, err := GenerateToken(1, "test_account", secretKey)
	if err != nil {
		fmt.Printf("生成token失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("生成的token: %s\n", token)
	if err := client.SendTokenVerify(token); err != nil {
		fmt.Printf("token验证失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("token验证请求已发送")

	// 等待1秒，让服务器处理token验证
	time.Sleep(1 * time.Second)

	// 3. 模拟玩家ID
	playerID := int64(1) // 固定玩家ID

	// 4. 测试进入地图
	fmt.Println("\n3. 测试进入地图...")
	if err := client.SendMapEnter(playerID, 1001); err != nil { // 1001是新手村地图ID
		fmt.Printf("进入地图失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("进入地图请求已发送: 角色ID=%d, 地图ID=1001\n", playerID)

	// 等待2秒，让服务器处理进入地图
	fmt.Println("\n等待服务器处理进入地图...")
	time.Sleep(2 * time.Second)

	// 5. 测试移动
	fmt.Println("\n4. 测试移动...")
	if err := client.SendMapMove(playerID, 1001, 255, 255, 0); err != nil { // 移动到坐标(255, 255, 0)
		fmt.Printf("移动失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("移动请求已发送: 角色ID=%d, 地图ID=1001, 坐标=(255, 255, 0)\n", playerID)

	// 等待1秒，让服务器处理移动
	fmt.Println("\n等待服务器处理移动...")
	time.Sleep(1 * time.Second)

	// 6. 测试攻击
	fmt.Println("\n5. 测试攻击...")
	if err := client.SendMapAttack(playerID, 1001, 10001); err != nil { // 攻击目标ID为10001的怪物
		fmt.Printf("攻击失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("攻击请求已发送: 角色ID=%d, 地图ID=1001, 目标ID=10001\n", playerID)

	// 等待1秒，让服务器处理攻击
	fmt.Println("\n等待服务器处理攻击...")
	time.Sleep(1 * time.Second)

	fmt.Println("\n测试完成!")
}

// checkPlayerInDatabase 检查数据库中是否有角色数据
// TokenClaims JWT声明
type TokenClaims struct {
	AccountID   int64  `json:"account_id"`
	AccountName string `json:"account_name"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT token
func GenerateToken(accountID int64, accountName, secretKey string) (string, error) {
	claims := &TokenClaims{
		AccountID:   accountID,
		AccountName: accountName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func checkPlayerInDatabase() {
	// 执行MySQL查询检查角色数据
	cmd := `mysql -h 192.168.91.128 -u root -ppotato -e "USE GameDB_000101; SELECT * FROM players;"`
	fmt.Printf("执行查询: %s\n", cmd)

	// 使用PowerShell执行命令
	output, err := exec.Command("powershell", "-Command", cmd).Output()
	if err != nil {
		fmt.Printf("查询数据库失败: %v\n", err)
		return
	}

	fmt.Println("数据库查询结果:")
	fmt.Println(string(output))

	// 检查是否有角色数据
	if strings.Contains(string(output), "testrole") {
		fmt.Println("\n✓ 角色数据已成功写入数据库!")
	} else {
		fmt.Println("\n✗ 角色数据未写入数据库!")
	}
}
