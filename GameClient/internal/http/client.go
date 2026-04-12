package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pzqf/zCommon/protocol"
)

// HTTP API 相关结构
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

type ServerListResponse struct {
	Result   int32                  `json:"result"`
	ErrorMsg string                 `json:"error_msg"`
	Servers  []*protocol.ServerInfo `json:"servers"`
}

// Client HTTP客户端
type Client struct {
	globalServerAddr string
}

// NewClient 创建新的HTTP客户端
func NewClient(globalServerAddr string) *Client {
	return &Client{
		globalServerAddr: globalServerAddr,
	}
}

// getServerAddr 返回完整的地址（地址:端口）
func getServerAddr(server *protocol.ServerInfo) string {
	return fmt.Sprintf("%s:%d", server.Address, server.Port)
}

// Login 登录
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

	return &authResp, nil
}

// Register 注册
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

// GetServerList 获取服务器列表
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

// SelectServer 选择服务器
func (c *Client) SelectServer(serverID int32, serverList []*protocol.ServerInfo) (*protocol.ServerInfo, string) {
	for _, server := range serverList {
		if server.ServerId == serverID {
			return server, getServerAddr(server)
		}
	}
	return nil, ""
}
