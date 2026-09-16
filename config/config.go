package config

import (
	"github.com/labstack/echo/v4"
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
}

type MySQLConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	Database string `json:"database" yaml:"database"`
}

type RedisConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
}

type WhitelistConfig struct {
	Health   []string `json:"health" yaml:"health"`
	Douyin   []string `json:"douyin" yaml:"douyin"`
	KuaiShou []string `json:"kuaishou" yaml:"kuaishou"`
	Internal []string `json:"internal" yaml:"internal"`
}

type Config struct {
	Server    *ServerConfig    `json:"server" yaml:"server"`
	Log       *LogConfig       `json:"log" yaml:"log"`
	MySQL     *MySQLConfig     `json:"mysql" yaml:"mysql"`
	Redis     *RedisConfig     `json:"redis" yaml:"redis"`
	Whitelist *WhitelistConfig `json:"whitelist" yaml:"whitelist"`
}

func Init(logger echo.Logger) (*Config, error) {
	viper.SetConfigFile("config.yaml")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		logger.Error("config.Init viper.ReadInConfig error! err[ %s ]", err.Error())
		return nil, err
	}
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		logger.Error("config.Init viper.Unmarshal error! err[ %s ]", err.Error())
		return nil, err
	}
	return cfg, nil
}
