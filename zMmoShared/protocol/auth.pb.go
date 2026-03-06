package protocol

// 账号相关协议
type AccountCreateRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

type AccountCreateResponse struct {
	Success  bool   `json:"success"`
	ErrorMsg string `json:"error_msg"`
}

type AccountLoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type AccountLoginResponse struct {
	Success bool         `json:"success"`
	ErrorMsg string       `json:"error_msg"`
	Token   string       `json:"token"`
	Servers []*ServerInfo `json:"servers"`
}

// 服务器相关协议
type ServerInfo struct {
	ServerId       int32  `json:"server_id"`
	ServerName     string `json:"server_name"`
	ServerType     string `json:"server_type"`
	GroupId        int32  `json:"group_id"`
	Address        string `json:"address"`
	Port           int32  `json:"port"`
	Status         int32  `json:"status"`
	OnlineCount    int32  `json:"online_count"`
	MaxOnlineCount int32  `json:"max_online_count"`
	Region         string `json:"region"`
	Version        string `json:"version"`
}

type ServerListResponse struct {
	Success  bool         `json:"success"`
	ErrorMsg string       `json:"error_msg"`
	Servers  []*ServerInfo `json:"servers"`
}

type ServerRegisterRequest struct {
	ServerId       int32  `json:"server_id"`
	ServerName     string `json:"server_name"`
	ServerType     string `json:"server_type"`
	GroupId        int32  `json:"group_id"`
	Address        string `json:"address"`
	Port           int32  `json:"port"`
	MaxOnlineCount int32  `json:"max_online_count"`
	Region         string `json:"region"`
	Version        string `json:"version"`
}

type ServerRegisterResponse struct {
	Success  bool   `json:"success"`
	ErrorMsg string `json:"error_msg"`
	ServerId int32  `json:"server_id"`
}

type ServerHeartbeatRequest struct {
	ServerId     int32 `json:"server_id"`
	OnlineCount  int32 `json:"online_count"`
	Status       int32 `json:"status"`
}

type ServerHeartbeatResponse struct {
	Success  bool   `json:"success"`
	ErrorMsg string `json:"error_msg"`
}
