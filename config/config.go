package config

import (
	"fmt"

	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	Host     string `json:"host" yaml:"host"`
	HTTPPort int    `json:"httpPort" yaml:"httpPort"`
	MQPort   int    `json:"mqPort" yaml:"mqPort"`
}

type LogConfig struct {
	Level   string `json:"level" yaml:"level"`
	LogFile string `json:"logFile" yaml:"logFile"`
	MaxSize int    `json:"maxSize" yaml:"maxSize"`
	MaxAge  int    `json:"maxAge" yaml:"maxAge"`
}

type MongoConfig struct {
	URI      string `json:"uri" yaml:"uri"`
	Database string `json:"database" yaml:"database"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`

	// 连接池
	PoolSize     int `json:"poolSize,omitempty" yaml:"poolSize,omitempty"`         // 最大连接数，默认 100
	MinIdleConns int `json:"minIdleConns,omitempty" yaml:"minIdleConns,omitempty"` // 最小空闲连接，默认 10
	MaxRetries   int `json:"maxRetries,omitempty" yaml:"maxRetries,omitempty"`     // 最大重试次数，默认 1

	// 超时（毫秒）
	DialTimeout  int `json:"dialTimeout,omitempty" yaml:"dialTimeout,omitempty"`   // 连接超时，默认 5000
	ReadTimeout  int `json:"readTimeout,omitempty" yaml:"readTimeout,omitempty"`   // 读超时，默认 3000
	WriteTimeout int `json:"writeTimeout,omitempty" yaml:"writeTimeout,omitempty"` // 写超时，默认 3000

	// 集群 + TLS
	IsCluster bool `json:"isCluster,omitempty" yaml:"isCluster,omitempty"` // 是否集群模式
	IsTLS     bool `json:"isTLS,omitempty" yaml:"isTLS,omitempty"`         // 是否启用 TLS
}

type NATSConfig struct {
	URL string `json:"url" yaml:"url"`
}

type Config struct {
	Server *ServerConfig `json:"server" yaml:"server"`
	Log    *LogConfig    `json:"log" yaml:"log"`
	Mongo  *MongoConfig  `json:"mongo" yaml:"mongo"`
	Redis  *RedisConfig  `json:"redis" yaml:"redis"`
	NATS   *NATSConfig   `json:"nats" yaml:"nats"`
}

// Init 初始化配置
func Init() (*Config, *codeerror.CodeError) {
	viper.SetConfigFile("config.yaml")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("config.Init viper.ReadInConfig error: %s\n", err.Error())
		return nil, codeerror.ConfigError.Msg("config.Init viper.ReadInConfig error: " + err.Error())
	}
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		fmt.Printf("config.Init viper.Unmarshal error: %s\n", err.Error())
		return nil, codeerror.ConfigError.Msg("config.Init viper.Unmarshal error: " + err.Error())
	}
	return cfg, nil
}
