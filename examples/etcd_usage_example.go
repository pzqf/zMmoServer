package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourusername/zMmoServer/zMmoShared/config"
	"github.com/yourusername/zMmoServer/zMmoShared/discovery"
)

func main() {
	// etcd 配置
	etcdEndpoints := []string{"http://localhost:2379"}
	etcdUsername := "root"
	etcdPassword := "potato"

	// 1. 服务注册示例
	log.Println("=== 服务注册示例 ===")
	serviceDiscovery, err := discovery.NewServiceDiscovery(etcdEndpoints, etcdUsername, etcdPassword)
	if err != nil {
		log.Fatalf("Failed to create service discovery: %v", err)
	}
	defer serviceDiscovery.Close()

	// 创建服务信息
	serviceInfo := &discovery.ServiceInfo{
		Name:     "gameserver",
		ID:       "000101",
		GroupID:  "1",
		Address:  "192.168.91.128",
		Port:     30001,
		Status:   "healthy",
		Load:     0.3,
		Players:  1234,
		Version:  "1.0.0",
		Metadata: map[string]string{
			"env":  "test",
			"zone": "zone1",
		},
	}

	// 注册服务
	ctx := context.Background()
	err = serviceDiscovery.Register(ctx, serviceInfo)
	if err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}
	log.Println("Service registered successfully")

	// 2. 服务发现示例
	log.Println("\n=== 服务发现示例 ===")
	services, err := serviceDiscovery.Discover(ctx, "gameserver", "1")
	if err != nil {
		log.Fatalf("Failed to discover services: %v", err)
	}

	log.Printf("Discovered %d services:\n", len(services))
	for _, svc := range services {
		log.Printf("Service: %s, ID: %s, Group: %s, Address: %s:%d, Status: %s, Load: %.2f, Players: %d\n",
			svc.Name, svc.ID, svc.GroupID, svc.Address, svc.Port, svc.Status, svc.Load, svc.Players)
	}

	// 3. 监听服务变化示例
	log.Println("\n=== 监听服务变化示例 ===")
	watchCtx, watchCancel := context.WithCancel(ctx)
	defer watchCancel()

	err = serviceDiscovery.Watch(watchCtx, "gameserver", "1", func(services []*discovery.ServiceInfo) {
		log.Printf("Service list updated, found %d services\n", len(services))
		for _, svc := range services {
			log.Printf("  - %s:%s:%s\n", svc.Name, svc.GroupID, svc.ID)
		}
	})
	if err != nil {
		log.Fatalf("Failed to watch services: %v", err)
	}

	// 4. 配置管理示例
	log.Println("\n=== 配置管理示例 ===")
	configManager, err := config.NewEtcdConfigManager(etcdEndpoints, etcdUsername, etcdPassword)
	if err != nil {
		log.Fatalf("Failed to create config manager: %v", err)
	}
	defer configManager.Close()

	// 设置配置
	configData := map[string]interface{}{
		"Server": map[string]interface{}{
			"ServerName": "GameServer",
			"ServerID":   1,
			"GroupID":    1,
			"ListenAddr": "0.0.0.0:20001",
		},
		"Database": map[string]interface{}{
			"DBHost":     "mysql",
			"DBPort":     3306,
			"DBName":     "GameDB_000101",
			"DBUser":     "root",
			"DBPassword": "potato",
		},
	}

	err = configManager.SetJSON(ctx, "gameserver", "1", "000101", "config", configData)
	if err != nil {
		log.Fatalf("Failed to set config: %v", err)
	}
	log.Println("Config set successfully")

	// 获取配置
	var retrievedConfig map[string]interface{}
	err = configManager.GetJSON(ctx, "gameserver", "1", "000101", "config", &retrievedConfig)
	if err != nil {
		log.Fatalf("Failed to get config: %v", err)
	}
	log.Printf("Retrieved config: %v\n", retrievedConfig)

	// 监听配置变化
	log.Println("\n=== 监听配置变化示例 ===")
	configWatchCtx, configWatchCancel := context.WithCancel(ctx)
	defer configWatchCancel()

	err = configManager.Watch(configWatchCtx, "gameserver", "1", "000101", "config", func(value string) {
		log.Printf("Config updated: %s\n", value)
	})
	if err != nil {
		log.Fatalf("Failed to watch config: %v", err)
	}

	// 模拟配置更新
	go func() {
		time.Sleep(2 * time.Second)
		updatedConfig := map[string]interface{}{
			"Server": map[string]interface{}{
				"ServerName": "GameServer",
				"ServerID":   1,
				"GroupID":    1,
				"ListenAddr": "0.0.0.0:20001",
				"LogLevel":   1,
			},
			"Database": map[string]interface{}{
				"DBHost":     "mysql",
				"DBPort":     3306,
				"DBName":     "GameDB_000101",
				"DBUser":     "root",
				"DBPassword": "potato",
			},
		}
		err := configManager.SetJSON(ctx, "gameserver", "1", "000101", "config", updatedConfig)
		if err != nil {
			log.Printf("Failed to update config: %v", err)
		} else {
			log.Println("Config updated successfully")
		}
	}()

	// 等待一段时间以观察变化
	time.Sleep(5 * time.Second)

	// 注销服务
	log.Println("\n=== 注销服务示例 ===")
	err = serviceDiscovery.Unregister(ctx, "gameserver", "1", "000101")
	if err != nil {
		log.Fatalf("Failed to unregister service: %v", err)
	}
	log.Println("Service unregistered successfully")

	log.Println("\nAll examples completed")
}