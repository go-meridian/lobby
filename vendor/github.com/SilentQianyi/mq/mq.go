package mq

import (
	"fmt"

	"github.com/SilentQianyi/logger"
)

// ClientFactory 客户端工厂函数类型
type ClientFactory func(cfg *Config) (MQClient, error)

// 全局工厂注册表
var factories = make(map[Mode]ClientFactory)

// Field 日志字段类型，别名 logger.Field
type Field = logger.Field

// RegisterFactory 注册客户端工厂函数
func RegisterFactory(mode Mode, factory ClientFactory) {
	factories[mode] = factory
}

// NewClient 根据配置创建 MQ 客户端
func NewClient(cfg *Config) (MQClient, error) {
	factory, ok := factories[cfg.Mode]
	if !ok {
		return nil, fmt.Errorf("unsupported mq mode: %s, please register factory first", cfg.Mode)
	}
	return factory(cfg)
}

// GetLogger 获取全局日志实例
func GetLogger() *logger.Logger {
	return logger.L()
}
