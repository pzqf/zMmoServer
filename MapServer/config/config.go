package config

import (
	"fmt"
	"strings"

	"github.com/pzqf/zCommon/common/id"
	cfgutil "github.com/pzqf/zCommon/config"
	"github.com/pzqf/zCommon/discovery"
	"github.com/pzqf/zCommon/metrics"
	"github.com/pzqf/zEngine/zLog"
	"github.com/pzqf/zUtil/zConfig"
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

	serverID := cfgutil.GetConfigInt(zcfg, "Server.ServerID", 1)

	c := &Config{
		Server: ServerConfig{
			ServerID:          serverID,
			ServerName:        cfgutil.GetConfigString(zcfg, "Server.ServerName", "MapServer"),
			GroupID:           cfgutil.GetConfigInt(zcfg, "Server.GroupID", 1),
			ListenAddr:        cfgutil.GetConfigString(zcfg, "Server.ListenAddr", "0.0.0.0:9002"),
			MaxConnections:    cfgutil.GetConfigInt(zcfg, "Server.MaxConnections", 10000),
			HeartbeatInterval: cfgutil.GetConfigInt(zcfg, "Server.HeartbeatInterval", 30),
		},
		Database: DatabaseConfig{
			DBType:          cfgutil.GetConfigString(zcfg, "Database.DBType", "mysql"),
			DBHost:          cfgutil.GetConfigString(zcfg, "Database.DBHost", "127.0.0.1"),
			DBPort:          cfgutil.GetConfigInt(zcfg, "Database.DBPort", 3306),
			DBName:          cfgutil.GetConfigString(zcfg, "Database.DBName", "MapDB"),
			DBUser:          cfgutil.GetConfigString(zcfg, "Database.DBUser", "root"),
			DBPassword:      cfgutil.GetConfigString(zcfg, "Database.DBPassword", ""),
			MaxOpenConns:    cfgutil.GetConfigInt(zcfg, "Database.MaxOpenConns", 100),
			MaxIdleConns:    cfgutil.GetConfigInt(zcfg, "Database.MaxIdleConns", 10),
			ConnMaxLifetime: cfgutil.GetConfigInt(zcfg, "Database.ConnMaxLifetime", 3600),
		},
		GameServer: GameServerConfig{
			GameServerAddr:           cfgutil.GetConfigString(zcfg, "GameServer.GameServerAddr", "127.0.0.1:20002"),
			GameServerConnectTimeout: cfgutil.GetConfigInt(zcfg, "GameServer.GameServerConnectTimeout", 10),
		},
		Log: zLog.Config{
			Level:              cfgutil.GetConfigInt(zcfg, "Log.Level", 0),
			Console:            cfgutil.GetConfigBool(zcfg, "Log.Console", true),
			ConsoleLevel:       cfgutil.GetConfigInt(zcfg, "Log.ConsoleLevel", 0),
			Filename:           cfgutil.ReplacePlaceholder(cfgutil.GetConfigString(zcfg, "Log.Filename", "./logs/map_server_{ServerID}.log"), "{ServerID}", serverID),
			MaxSize:            cfgutil.GetConfigInt(zcfg, "Log.MaxSize", 100),
			MaxDays:            cfgutil.GetConfigInt(zcfg, "Log.MaxDays", 15),
			MaxBackups:         cfgutil.GetConfigInt(zcfg, "Log.MaxBackups", 10),
			Compress:           cfgutil.GetConfigBool(zcfg, "Log.Compress", true),
			ShowCaller:         cfgutil.GetConfigBool(zcfg, "Log.ShowCaller", true),
			Stacktrace:         cfgutil.GetConfigInt(zcfg, "Log.Stacktrace", 3),
			Sampling:           cfgutil.GetConfigBool(zcfg, "Log.Sampling", true),
			SamplingInitial:    cfgutil.GetConfigInt(zcfg, "Log.SamplingInitial", 100),
			SamplingThereafter: cfgutil.GetConfigInt(zcfg, "Log.SamplingThereafter", 10),
			Async:              cfgutil.GetConfigBool(zcfg, "Log.Async", true),
			AsyncBufferSize:    cfgutil.GetConfigInt(zcfg, "Log.AsyncBufferSize", 2048),
			AsyncFlushInterval: cfgutil.GetConfigInt(zcfg, "Log.AsyncFlushInterval", 50),
		},
		Metrics: MetricsConfig{
			Enabled:       cfgutil.GetConfigBool(zcfg, "Metrics.Enabled", true),
			ListenAddress: cfgutil.GetConfigString(zcfg, "Metrics.ListenAddress", "0.0.0.0:9093"),
		},
		Pprof: PprofConfig{
			Enabled:       cfgutil.GetConfigBool(zcfg, "Pprof.Enabled", false),
			ListenAddress: cfgutil.GetConfigString(zcfg, "Pprof.ListenAddress", "localhost:6063"),
		},
		Etcd: discovery.EtcdConfig{
			Endpoints:      cfgutil.GetConfigString(zcfg, "Etcd.Endpoints", "etcd-cluster.kube-system.svc.cluster.local:2379"),
			Username:       cfgutil.GetConfigString(zcfg, "Etcd.Username", ""),
			Password:       cfgutil.GetConfigString(zcfg, "Etcd.Password", ""),
			CACertPath:     cfgutil.GetConfigString(zcfg, "Etcd.CACertPath", "../resources/etcd/ca.crt"),
			ClientCertPath: cfgutil.GetConfigString(zcfg, "Etcd.ClientCertPath", "../resources/etcd/server.crt"),
			ClientKeyPath:  cfgutil.GetConfigString(zcfg, "Etcd.ClientKeyPath", "../resources/etcd/server.key"),
		},
		Maps: MapsConfig{
			Mode:   strings.ToLower(cfgutil.GetConfigString(zcfg, "Maps.Mode", MapModeSingleServer)),
			MapIDs: cfgutil.GetConfigIntSlice(zcfg, "Maps.MapIDs", []int{1001}),
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
