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
			Mode:   strings.ToLower(zConfig.GetStringWithDefault(zcfg, "Maps.Mode", MapModeSingleServer)),
			MapIDs: zConfig.GetIntSliceWithDefault(zcfg, "Maps.MapIDs", []int{1001}),
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
	return nil
}

func (c *Config) GetLogConfig() *zLog.Config {
	return &c.Log
}
