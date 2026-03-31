package configcenter

import (
	"context"
)

// ConfigCenter 配置中心接口
type ConfigCenter interface {
	// Get 获取配置
	Get(ctx context.Context, key string) (string, error)

	// Set 设置配置
	Set(ctx context.Context, key string, value string) error

	// Delete 删除配置
	Delete(ctx context.Context, key string) error

	// Watch 监听配置变化
	Watch(ctx context.Context, key string, callback func(string, string)) error

	// Close 关闭配置中心
	Close() error
}

// NewConfigCenter 创建配置中心实例
func NewConfigCenter(endpoints []string, username, password string) (ConfigCenter, error) {
	return NewEtcdConfigCenter(endpoints, username, password), nil
}
