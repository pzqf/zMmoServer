package models

import (
	"time"
)

type GameServer struct {
	ServerID       int32     `gorm:"column:server_id;primaryKey;autoIncrement" json:"serverId"`
	ServerName     string    `gorm:"column:server_name;size:255;not null" json:"serverName"`
	ServerType     string    `gorm:"column:server_type;size:50;not null" json:"serverType"`
	GroupID        int32     `gorm:"column:group_id;default:0" json:"groupId"`
	Address        string    `gorm:"column:address;size:255;not null" json:"address"`
	Port           int32     `gorm:"column:port;not null" json:"port"`
	Status         int32     `gorm:"column:status;not null" json:"status"`
	OnlineCount    int32     `gorm:"column:online_count;default:0" json:"onlineCount"`
	MaxOnlineCount int32     `gorm:"column:max_online_count;default:5000" json:"maxOnlineCount"`
	Region         string    `gorm:"column:region;size:100" json:"region"`
	Version        string    `gorm:"column:version;size:50" json:"version"`
	LastHeartbeat  time.Time `gorm:"column:last_heartbeat;autoCreateTime" json:"lastHeartbeat"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (m GameServer) TableName() string {
	return "game_servers"
}

func (m GameServer) GetTableName() string {
	return "game_servers"
}

func (m *GameServer) GetModel() interface{} {
	return m
}

func (m *GameServer) GetID() interface{} {
	return m.ServerID
}

func (m *GameServer) SetID(id interface{}) {
	if val, ok := id.(int32); ok {
		m.ServerID = val
	}
}
