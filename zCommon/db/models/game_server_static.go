package models

import (
	"time"
)

// GameServerStatic 游戏服务器静态配置（存储在MySQL）
// 这些字段由运维管理，很少变动
type GameServerStatic struct {
	ServerID       int32     `gorm:"column:server_id;primaryKey" json:"serverId"`
	ServerName     string    `gorm:"column:server_name;size:255;not null" json:"serverName"`
	ServerType     string    `gorm:"column:server_type;size:50;not null" json:"serverType"`
	GroupID        int32     `gorm:"column:group_id;default:0" json:"groupId"`
	MaxOnlineCount int32     `gorm:"column:max_online_count;default:5000" json:"maxOnlineCount"`
	Region         string    `gorm:"column:region;size:100" json:"region"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (m GameServerStatic) TableName() string {
	return "game_servers"
}

func (m GameServerStatic) GetTableName() string {
	return "game_servers"
}

func (m *GameServerStatic) GetModel() interface{} {
	return m
}

func (m *GameServerStatic) GetID() interface{} {
	return m.ServerID
}

func (m *GameServerStatic) SetID(id interface{}) {
	if val, ok := id.(int32); ok {
		m.ServerID = val
	}
}
