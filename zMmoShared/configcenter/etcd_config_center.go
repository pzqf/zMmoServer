package configcenter

import (
	"context"
	"fmt"
	"time"

	"go.etcd.io/etcd/client/v3"
)

// EtcdConfigCenter etcd配置中心实现
type EtcdConfigCenter struct {
	client *clientv3.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// NewEtcdConfigCenter 创建etcd配置中心实例
func NewEtcdConfigCenter(endpoints []string, username, password string) (*EtcdConfigCenter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
		Username:    username,
		Password:    password,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &EtcdConfigCenter{
		client: client,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Get 获取配置
func (c *EtcdConfigCenter) Get(ctx context.Context, key string) (string, error) {
	resp, err := c.client.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return "", nil
	}

	return string(resp.Kvs[0].Value), nil
}

// Set 设置配置
func (c *EtcdConfigCenter) Set(ctx context.Context, key string, value string) error {
	_, err := c.client.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}

	return nil
}

// Delete 删除配置
func (c *EtcdConfigCenter) Delete(ctx context.Context, key string) error {
	_, err := c.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete config: %w", err)
	}

	return nil
}

// Watch 监听配置变化
func (c *EtcdConfigCenter) Watch(ctx context.Context, key string, callback func(string, string)) error {
	// 创建监听器
	watcher := c.client.Watch(ctx, key)

	// 处理变更
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.ctx.Done():
				return
			case resp, ok := <-watcher:
				if !ok {
					return
				}

				for _, event := range resp.Events {
					callback(string(event.Kv.Key), string(event.Kv.Value))
				}
			}
		}
	}()

	return nil
}

// Close 关闭配置中心
func (c *EtcdConfigCenter) Close() error {
	c.cancel()
	return c.client.Close()
}
