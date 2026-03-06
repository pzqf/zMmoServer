package constants

// ServiceID 服务ID定义
type ServiceID int32

const (
	// 全局服务 (0-99)
	GlobalService ServiceID = 0  // 全局服务

	// 网关服务 (100-199)
	GatewayService ServiceID = 100  // 网关服务

	// 游戏逻辑服务 (200-299)
	GameService1 ServiceID = 200  // 游戏服务1
	GameService2 ServiceID = 201  // 游戏服务2
	GameService3 ServiceID = 202  // 游戏服务3
	GameService4 ServiceID = 203  // 游戏服务4
	GameService5 ServiceID = 204  // 游戏服务5

	// 地图服务 (300-399)
	MapService1 ServiceID = 300  // 地图服务1
	MapService2 ServiceID = 301  // 地图服务2
	MapService3 ServiceID = 302  // 地图服务3

	// 后台管理服务 (400-499)
	AdminService ServiceID = 400  // 后台管理服务
)

// IsValidServiceID 检查服务ID是否有效
func IsValidServiceID(id ServiceID) bool {
	return id >= 0 && id <= 999
}

// IsGatewayService 检查是否是网关服务
func IsGatewayService(id ServiceID) bool {
	return id >= 100 && id < 200
}

// IsGameService 检查是否是游戏服务
func IsGameService(id ServiceID) bool {
	return id >= 200 && id < 300
}

// IsMapService 检查是否是地图服务
func IsMapService(id ServiceID) bool {
	return id >= 300 && id < 400
}

// IsAdminService 检查是否是后台管理服务
func IsAdminService(id ServiceID) bool {
	return id >= 400 && id < 500
}
