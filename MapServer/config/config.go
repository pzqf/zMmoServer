package config

import (
	"fmt"
	"strings"

	"github.com/pzqf/zCommon/common/id"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zConfig"
	"github.com/pzqf/zEngine/zLog"
)

const (
	MapModeSingleServer = "single_server"
	MapModeMirror       = "mirror"
	MapModeCrossGroup   = "cross_group"
)

type Config struct {
	Server     ServerConfig         `ini:"Server"`
	Database   DatabaseConfig       `ini:"Database"`
	GameServer GameServerConfig     `ini:"GameServer"`
	Etcd       discovery.EtcdConfig `ini:"Etcd"`
	Log        zLog.Config          `ini:"Log"`
	Metrics    MetricsConfig        `ini:"Metrics"`
	Pprof      PprofConfig          `ini:"Pprof"`
	Maps       MapsConfig           `ini:"Maps"`
}

type MapsConfig struct {
	Mode   string `ini:"Mode"`
	MapIDs []int  `ini:"MapIDs"`
	// 分线/无缝的生产配置管线（原来 RegisterLayerable 仅 ZMMO_TEST_LAYER 测试夹具、
	// RegisterSeamlessLink 零生产接线；这里由 ini 驱动，让分线/无缝拓扑成为配置而非测试夹具）。
	LayerableMaps []int  `ini:"LayerableMaps"` // 可分线的逻辑地图ID子集（空=不分线）
	LayerSoftCap  int    `ini:"LayerSoftCap"`  // 单层软上限：到达后新玩家开新层
	LayerHardCap  int    `ini:"LayerHardCap"`  // 单层硬上限：亲和也不能超过
	SeamlessLinks string `ini:"SeamlessLinks"` // 无缝相邻图对，格式 "a-b,c-d"（双向登记）
}

// ParseSeamlessLinks 解析 SeamlessLinks（"a-b,c-d"）为图对；非法项跳过。
func (mc *MapsConfig) ParseSeamlessLinks() [][2]int32 {
	var pairs [][2]int32
	if strings.TrimSpace(mc.SeamlessLinks) == "" {
		return pairs
	}
	for _, seg := range strings.Split(mc.SeamlessLinks, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		var a, b int32
		if _, err := fmt.Sscanf(seg, "%d-%d", &a, &b); err != nil || a == 0 || b == 0 {
			continue
		}
		pairs = append(pairs, [2]int32{a, b})
	}
	return pairs
}

// IsLayerableConfigured 该逻辑地图是否在 ini 中声明为可分线。
func (mc *MapsConfig) IsLayerableConfigured(mapID int32) bool {
	for _, m := range mc.LayerableMaps {
		if int32(m) == mapID {
			return true
		}
	}
	return false
}

type ServerConfig struct {
	ServerID          int    `ini:"ServerID"`
	ServerName        string `ini:"ServerName"`
	GroupID           int    `ini:"GroupID"`
	ListenAddr        string `ini:"ListenAddr"`
	MaxConnections    int    `ini:"MaxConnections"`
	HeartbeatInterval int    `ini:"HeartbeatInterval"`
}

type DatabaseConfig struct {
	DBType          string `ini:"DBType"`
	DBHost          string `ini:"DBHost"`
	DBPort          int    `ini:"DBPort"`
	DBName          string `ini:"DBName"`
	DBUser          string `ini:"DBUser"`
	DBPassword      string `ini:"DBPassword"`
	MaxOpenConns    int    `ini:"MaxOpenConns"`
	MaxIdleConns    int    `ini:"MaxIdleConns"`
	ConnMaxLifetime int    `ini:"ConnMaxLifetime"`
}

type GameServerConfig struct {
	GameServerAddr           string `ini:"GameServerAddr"`
	GameServerConnectTimeout int    `ini:"GameServerConnectTimeout"`
}

type MetricsConfig metrics.MetricsConfig

type PprofConfig struct {
	Enabled       bool   `ini:"Enabled"`
	ListenAddress string `ini:"ListenAddress"`
}

func LoadConfig(configPath string) (*Config, error) {
	zcfg := zConfig.NewConfig()
	if err := zcfg.LoadINI(configPath); err != nil {
		return nil, fmt.Errorf("failed to load config file: %v", err)
	}

	serverID := zConfig.GetIntWithDefault(zcfg, "Server.ServerID", 1)

	c := &Config{
		Server: ServerConfig{
			ServerID:          serverID,
			ServerName:        zConfig.GetStringWithDefault(zcfg, "Server.ServerName", "MapServer"),
			GroupID:           zConfig.GetIntWithDefault(zcfg, "Server.GroupID", 1),
			ListenAddr:        zConfig.GetStringWithDefault(zcfg, "Server.ListenAddr", "0.0.0.0:9002"),
			MaxConnections:    zConfig.GetIntWithDefault(zcfg, "Server.MaxConnections", 10000),
			HeartbeatInterval: zConfig.GetIntWithDefault(zcfg, "Server.HeartbeatInterval", 30),
		},
		Database: DatabaseConfig{
			DBType:          zConfig.GetStringWithDefault(zcfg, "Database.DBType", "mysql"),
			DBHost:          zConfig.GetStringWithDefault(zcfg, "Database.DBHost", "127.0.0.1"),
			DBPort:          zConfig.GetIntWithDefault(zcfg, "Database.DBPort", 3306),
			DBName:          zConfig.GetStringWithDefault(zcfg, "Database.DBName", "MapDB"),
			DBUser:          zConfig.GetStringWithDefault(zcfg, "Database.DBUser", "root"),
			DBPassword:      zConfig.GetStringWithDefault(zcfg, "Database.DBPassword", ""),
			MaxOpenConns:    zConfig.GetIntWithDefault(zcfg, "Database.MaxOpenConns", 100),
			MaxIdleConns:    zConfig.GetIntWithDefault(zcfg, "Database.MaxIdleConns", 10),
			ConnMaxLifetime: zConfig.GetIntWithDefault(zcfg, "Database.ConnMaxLifetime", 3600),
		},
		GameServer: GameServerConfig{
			GameServerAddr:           zConfig.GetStringWithDefault(zcfg, "GameServer.GameServerAddr", "127.0.0.1:20002"),
			GameServerConnectTimeout: zConfig.GetIntWithDefault(zcfg, "GameServer.GameServerConnectTimeout", 10),
		},
		Log: zLog.Config{
			Level:              zConfig.GetIntWithDefault(zcfg, "Log.Level", 0),
			Console:            zConfig.GetBoolWithDefault(zcfg, "Log.Console", true),
			ConsoleLevel:       zConfig.GetIntWithDefault(zcfg, "Log.ConsoleLevel", 0),
			Filename:           zConfig.ReplacePlaceholder(zConfig.GetStringWithDefault(zcfg, "Log.Filename", "./logs/map_server_{ServerID}.log"), "{ServerID}", serverID),
			MaxSize:            zConfig.GetIntWithDefault(zcfg, "Log.MaxSize", 100),
			MaxDays:            zConfig.GetIntWithDefault(zcfg, "Log.MaxDays", 15),
			MaxBackups:         zConfig.GetIntWithDefault(zcfg, "Log.MaxBackups", 10),
			Compress:           zConfig.GetBoolWithDefault(zcfg, "Log.Compress", true),
			ShowCaller:         zConfig.GetBoolWithDefault(zcfg, "Log.ShowCaller", true),
			Stacktrace:         zConfig.GetIntWithDefault(zcfg, "Log.Stacktrace", 3),
			Sampling:           zConfig.GetBoolWithDefault(zcfg, "Log.Sampling", true),
			SamplingInitial:    zConfig.GetIntWithDefault(zcfg, "Log.SamplingInitial", 100),
			SamplingThereafter: zConfig.GetIntWithDefault(zcfg, "Log.SamplingThereafter", 10),
			Async:              zConfig.GetBoolWithDefault(zcfg, "Log.Async", true),
			AsyncBufferSize:    zConfig.GetIntWithDefault(zcfg, "Log.AsyncBufferSize", 2048),
			AsyncFlushInterval: zConfig.GetIntWithDefault(zcfg, "Log.AsyncFlushInterval", 50),
		},
		Metrics: MetricsConfig{
			Enabled:       zConfig.GetBoolWithDefault(zcfg, "Metrics.Enabled", true),
			ListenAddress: zConfig.GetStringWithDefault(zcfg, "Metrics.ListenAddress", "0.0.0.0:9093"),
		},
		Pprof: PprofConfig{
			Enabled:       zConfig.GetBoolWithDefault(zcfg, "Pprof.Enabled", false),
			ListenAddress: zConfig.GetStringWithDefault(zcfg, "Pprof.ListenAddress", "localhost:6063"),
		},
		Etcd: discovery.EtcdConfig{
			Endpoints:      zConfig.GetStringWithDefault(zcfg, "Etcd.Endpoints", "etcd-cluster.kube-system.svc.cluster.local:2379"),
			Username:       zConfig.GetStringWithDefault(zcfg, "Etcd.Username", ""),
			Password:       zConfig.GetStringWithDefault(zcfg, "Etcd.Password", ""),
			CACertPath:     zConfig.GetStringWithDefault(zcfg, "Etcd.CACertPath", "../resources/etcd/ca.crt"),
			ClientCertPath: zConfig.GetStringWithDefault(zcfg, "Etcd.ClientCertPath", "../resources/etcd/server.crt"),
			ClientKeyPath:  zConfig.GetStringWithDefault(zcfg, "Etcd.ClientKeyPath", "../resources/etcd/server.key"),
		},
		Maps: MapsConfig{
			Mode:          strings.ToLower(zConfig.GetStringWithDefault(zcfg, "Maps.Mode", MapModeSingleServer)),
			MapIDs:        zConfig.GetIntSliceWithDefault(zcfg, "Maps.MapIDs", []int{1001}),
			LayerableMaps: zConfig.GetIntSliceWithDefault(zcfg, "Maps.LayerableMaps", nil),
			LayerSoftCap:  zConfig.GetIntWithDefault(zcfg, "Maps.LayerSoftCap", 0),
			LayerHardCap:  zConfig.GetIntWithDefault(zcfg, "Maps.LayerHardCap", 0),
			SeamlessLinks: zConfig.GetStringWithDefault(zcfg, "Maps.SeamlessLinks", ""),
		},
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Config) Validate() error {
	if _, err := id.ParseServerIDInt(int32(c.Server.ServerID)); err != nil {
		return fmt.Errorf("invalid ServerID %d: %w", c.Server.ServerID, err)
	}
	if c.Server.ListenAddr == "" {
		return fmt.Errorf("Server.ListenAddr is required")
	}
	switch strings.ToLower(c.Maps.Mode) {
	case MapModeSingleServer, MapModeMirror, MapModeCrossGroup:
	default:
		return fmt.Errorf(
			"invalid Maps.Mode %q, allowed values: %s, %s, %s",
			c.Maps.Mode,
			MapModeSingleServer,
			MapModeMirror,
			MapModeCrossGroup,
		)
	}
	c.Maps.Mode = strings.ToLower(c.Maps.Mode)

	if len(c.Maps.MapIDs) == 0 {
		return fmt.Errorf("Maps.MapIDs must not be empty")
	}
	// 声明了可分线图就必须给出合法的软/硬上限，否则分线无从执行（分线是 realm 承重件，不容静默失效）。
	if len(c.Maps.LayerableMaps) > 0 {
		if c.Maps.LayerSoftCap <= 0 || c.Maps.LayerHardCap <= 0 {
			return fmt.Errorf("Maps.LayerableMaps set but LayerSoftCap/LayerHardCap not positive")
		}
		if c.Maps.LayerSoftCap > c.Maps.LayerHardCap {
			return fmt.Errorf("Maps.LayerSoftCap(%d) must be <= LayerHardCap(%d)", c.Maps.LayerSoftCap, c.Maps.LayerHardCap)
		}
	}
	return nil
}

func (c *Config) GetLogConfig() *zLog.Config {
	return &c.Log
}
