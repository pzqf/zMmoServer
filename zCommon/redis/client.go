package redis

import (
	"context"
	"fmt"
	"time"

	redisv8 "github.com/go-redis/redis/v8"
	"github.com/pzqf/zEngine/zLog"
	"go.uber.org/zap"
)

// Client Redis客户端
type Client struct {
	client *redisv8.Client
	ctx    context.Context
}

// RedisConfig Redis配置 - 统一配置格式，便于各服务使用
type RedisConfig struct {
	Host     string `ini:"host" json:"host"`
	Port     int    `ini:"port" json:"port"`
	Password string `ini:"password" json:"password"`
	DB       int    `ini:"db" json:"db"`
	PoolSize int    `ini:"pool_size" json:"pool_size"`
}

// DefaultConfig 默认配置
func DefaultConfig() RedisConfig {
	return RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
		PoolSize: 10,
	}
}

// NewClient 创建Redis客户端
func NewClient(cfg RedisConfig) (*Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	rdb := redisv8.NewClient(&redisv8.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx := context.Background()

	// 测试连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	zLog.Info("Redis client connected",
		zap.String("addr", addr),
		zap.Int("db", cfg.DB),
	)

	return &Client{
		client: rdb,
		ctx:    ctx,
	}, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// GetClient 获取原生Redis客户端
func (c *Client) GetClient() *redisv8.Client {
	return c.client
}

// GetContext 获取上下文
func (c *Client) GetContext() context.Context {
	return c.ctx
}

// HSet 设置Hash字段
func (c *Client) HSet(key string, values ...interface{}) error {
	return c.client.HSet(c.ctx, key, values...).Err()
}

// HGet 获取Hash字段
func (c *Client) HGet(key, field string) (string, error) {
	return c.client.HGet(c.ctx, key, field).Result()
}

// HGetAll 获取所有Hash字段
func (c *Client) HGetAll(key string) (map[string]string, error) {
	return c.client.HGetAll(c.ctx, key).Result()
}

// HDel 删除Hash字段
func (c *Client) HDel(key string, fields ...string) error {
	return c.client.HDel(c.ctx, key, fields...).Err()
}

// Del 删除Key
func (c *Client) Del(keys ...string) error {
	return c.client.Del(c.ctx, keys...).Err()
}

// Keys 查找Key
func (c *Client) Keys(pattern string) ([]string, error) {
	return c.client.Keys(c.ctx, pattern).Result()
}

// Expire 设置过期时间
func (c *Client) Expire(key string, expiration time.Duration) error {
	return c.client.Expire(c.ctx, key, expiration).Err()
}

// ZAdd 添加有序集合成员
func (c *Client) ZAdd(key string, members ...*redisv8.Z) error {
	return c.client.ZAdd(c.ctx, key, members...).Err()
}

// ZRange 获取有序集合范围
func (c *Client) ZRange(key string, start, stop int64) ([]string, error) {
	return c.client.ZRange(c.ctx, key, start, stop).Result()
}

// ZRem 删除有序集合成员
func (c *Client) ZRem(key string, members ...interface{}) error {
	return c.client.ZRem(c.ctx, key, members...).Err()
}

// SAdd 添加集合成员
func (c *Client) SAdd(key string, members ...interface{}) error {
	return c.client.SAdd(c.ctx, key, members...).Err()
}

// SMembers 获取集合所有成员
func (c *Client) SMembers(key string) ([]string, error) {
	return c.client.SMembers(c.ctx, key).Result()
}

// SRem 删除集合成员
func (c *Client) SRem(key string, members ...interface{}) error {
	return c.client.SRem(c.ctx, key, members...).Err()
}

// Set 设置字符串值
func (c *Client) Set(key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(c.ctx, key, value, expiration).Err()
}

// Get 获取字符串值
func (c *Client) Get(key string) (string, error) {
	return c.client.Get(c.ctx, key).Result()
}
