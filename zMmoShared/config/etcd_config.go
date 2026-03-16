package config

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdConfigManager etcd配置管理器
type EtcdConfigManager struct {
	client *clientv3.Client
}

// NewEtcdConfigManager 创建etcd配置管理器实例
func NewEtcdConfigManager(endpoints []string, username, password string) (*EtcdConfigManager, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		Username:    username,
		Password:    password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &EtcdConfigManager{
		client: client,
	}, nil
}

// Get 获取配置
func (c *EtcdConfigManager) Get(ctx context.Context, serviceType, groupID, serviceID, key string) (string, error) {
	configPath := filepath.Join("config", serviceType, groupID, serviceID, key)
	resp, err := c.client.Get(ctx, configPath)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("config not found: %s", configPath)
	}

	return string(resp.Kvs[0].Value), nil
}

// GetJSON 获取JSON配置并解析为指定类型
func (c *EtcdConfigManager) GetJSON(ctx context.Context, serviceType, groupID, serviceID, key string, v interface{}) error {
	value, err := c.Get(ctx, serviceType, groupID, serviceID, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(value), v)
}

// Set 设置配置
func (c *EtcdConfigManager) Set(ctx context.Context, serviceType, groupID, serviceID, key, value string) error {
	configPath := filepath.Join("config", serviceType, groupID, serviceID, key)
	_, err := c.client.Put(ctx, configPath, value)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	return nil
}

// SetJSON 设置JSON配置
func (c *EtcdConfigManager) SetJSON(ctx context.Context, serviceType, groupID, serviceID, key string, v interface{}) error {
	value, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return c.Set(ctx, serviceType, groupID, serviceID, key, string(value))
}

// Delete 删除配置
func (c *EtcdConfigManager) Delete(ctx context.Context, serviceType, groupID, serviceID, key string) error {
	configPath := filepath.Join("config", serviceType, groupID, serviceID, key)
	_, err := c.client.Delete(ctx, configPath)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	return nil
}

// Watch 监听配置变化
func (c *EtcdConfigManager) Watch(ctx context.Context, serviceType, groupID, serviceID, key string, callback func(string)) error {
	configPath := filepath.Join("config", serviceType, groupID, serviceID, key)
	watcher := c.client.Watch(ctx, configPath)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case resp, ok := <-watcher:
				if !ok {
					return
				}

				for _, event := range resp.Events {
					if event.Type == clientv3.EventTypePut {
						callback(string(event.Kv.Value))
					}
				}
			}
		}
	}()

	return nil
}

// Close 关闭配置管理器
func (c *EtcdConfigManager) Close() error {
	return c.client.Close()
}