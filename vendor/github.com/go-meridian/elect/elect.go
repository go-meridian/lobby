package elect

import (
	"fmt"
	"os"
)

// Elector 选举器接口
type Elector interface {
	// IsLeader 当前实例是否为 Leader
	IsLeader() bool

	// LeaderID 当前 Leader 的标识
	LeaderID() string

	// Close 优雅退出
	Close()
}

// Mode 选举模式
type Mode string

const (
	ModeEtcd  Mode = "etcd"
	ModeRedis Mode = "redis"
)

// Config 选举配置
type Config struct {
	Mode     Mode         `json:"mode" yaml:"mode"`
	Etcd     *EtcdConfig  `json:"etcd,omitempty" yaml:"etcd,omitempty"`
	Redis    *RedisConfig `json:"redis,omitempty" yaml:"redis,omitempty"`
	Prefix   string       `json:"prefix,omitempty" yaml:"prefix,omitempty"`       // 选主 key 前缀，默认 "/elect/leader"
	TTL      int          `json:"ttl,omitempty" yaml:"ttl,omitempty"`              // 租约 TTL 秒，默认 10
	LeaderID string       `json:"-" yaml:"-"`                                      // 当前实例 ID，自动生成
}

// EtcdConfig etcd 配置
type EtcdConfig struct {
	Endpoints   []string `json:"endpoints" yaml:"endpoints"`
	Username    string   `json:"username,omitempty" yaml:"username,omitempty"`
	Password    string   `json:"password,omitempty" yaml:"password,omitempty"`
	DialTimeout int      `json:"dialTimeout,omitempty" yaml:"dialTimeout,omitempty"` // 毫秒，默认 5000
}

// RedisConfig redis 配置
type RedisConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	DB       int    `json:"db,omitempty" yaml:"db,omitempty"`
}

// ElectorFactory 选举器工厂函数
type ElectorFactory func(cfg *Config, opts *Options) (Elector, error)

// 全局工厂注册表
var factories = make(map[Mode]ElectorFactory)

// RegisterFactory 注册选举器工厂
func RegisterFactory(mode Mode, factory ElectorFactory) {
	factories[mode] = factory
}

// New 根据配置创建选举器
func New(cfg *Config, opts ...Option) (Elector, error) {
	factory, ok := factories[cfg.Mode]
	if !ok {
		return nil, fmt.Errorf("unsupported elect mode: %s", cfg.Mode)
	}

	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}

	// 自动生成 LeaderID
	if cfg.LeaderID == "" {
		hostname, _ := os.Hostname()
		cfg.LeaderID = fmt.Sprintf("%s_%d", hostname, os.Getpid())
	}

	return factory(cfg, o)
}

// OrDefault 返回 val，若 val <= 0 则返回 defaultVal
func OrDefault(val, defaultVal int) int {
	if val <= 0 {
		return defaultVal
	}
	return val
}

// OrDefaultStr 返回 val，若 val 为空则返回 defaultVal
func OrDefaultStr(val, defaultVal string) string {
	if val == "" {
		return defaultVal
	}
	return val
}
