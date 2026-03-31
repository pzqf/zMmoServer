package configcenter

import (
	"context"
	"fmt"
	"time"

	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// EtcdConfigCenter 基于 etcd 的配置中心实现（适配 k8s）
type EtcdConfigCenter struct {
	// 这里可以添加 etcd 客户端
	// 由于 etcd 在 k8s 集群内，可以通过服务名访问
	endpoints []string
	username  string
	password  string
}

// NewEtcdConfigCenter 创建 etcd 配置中心
func NewEtcdConfigCenter(endpoints []string, username, password string) *EtcdConfigCenter {
	zLog.Info("Creating EtcdConfigCenter for k8s cluster", 
		zap.Strings("endpoints", endpoints))
	
	return &EtcdConfigCenter{
		endpoints: endpoints,
		username:  username,
		password:  password,
	}
}

// Get 获取配置值
func (e *EtcdConfigCenter) Get(ctx context.Context, key string) (string, error) {
	// TODO: 实现 etcd 客户端调用
	zLog.Info("Getting config from etcd", zap.String("key", key))
	return "", fmt.Errorf("etcd client not implemented yet")
}

// Set 设置配置值
func (e *EtcdConfigCenter) Set(ctx context.Context, key, value string) error {
	// TODO: 实现 etcd 客户端调用
	zLog.Info("Setting config to etcd", 
		zap.String("key", key),
		zap.String("value", value))
	return fmt.Errorf("etcd client not implemented yet")
}

// Delete 删除配置值
func (e *EtcdConfigCenter) Delete(ctx context.Context, key string) error {
	// TODO: 实现 etcd 客户端调用
	zLog.Info("Deleting config from etcd", zap.String("key", key))
	return fmt.Errorf("etcd client not implemented yet")
}

// Watch 监听配置变化
func (e *EtcdConfigCenter) Watch(ctx context.Context, key string, callback func(string, string)) error {
	// TODO: 实现 etcd 客户端调用
	zLog.Info("Watching config from etcd", zap.String("key", key))
	return fmt.Errorf("etcd client not implemented yet")
}

// Close 关闭配置中心
func (e *EtcdConfigCenter) Close() error {
	zLog.Info("Closing EtcdConfigCenter")
	return nil
}

// WatchWithRetry 带重试的监听
func (e *EtcdConfigCenter) WatchWithRetry(ctx context.Context, key string, callback func(string, string), maxRetries int, retryInterval time.Duration) error {
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := e.Watch(ctx, key, callback)
			if err == nil {
				return nil
			}
			
			lastErr = err
			zLog.Warn("Watch failed, retrying", 
				zap.Int("attempt", i+1),
				zap.Int("max_retries", maxRetries),
				zap.Duration("retry_interval", retryInterval),
				zap.Error(err))
			
			if i < maxRetries-1 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(retryInterval):
					// 继续重试
				}
			}
		}
	}
	
	return fmt.Errorf("watch failed after %d retries: %w", maxRetries, lastErr)
}
