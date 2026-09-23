package dao

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-meridian/lobby/config"
	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/logger"
	"github.com/redis/go-redis/v9"
)

// RDB Redis 客户端实例
var RDB redis.Cmdable

// Init 初始化 Redis
func Init(cfg *config.Config) *codeerror.CodeError {
	addr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	poolSize := orDefault(cfg.Redis.PoolSize, 100)
	minIdle := orDefault(cfg.Redis.MinIdleConns, 10)
	maxRetries := orDefault(cfg.Redis.MaxRetries, 1)
	dialTimeout := time.Duration(orDefault(cfg.Redis.DialTimeout, 5000)) * time.Millisecond
	readTimeout := time.Duration(orDefault(cfg.Redis.ReadTimeout, 3000)) * time.Millisecond
	writeTimeout := time.Duration(orDefault(cfg.Redis.WriteTimeout, 3000)) * time.Millisecond
	tlsCfg := buildTLSConfig(cfg.Redis.IsTLS)

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	if cfg.Redis.IsCluster {
		client := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        []string{addr},
			Password:     cfg.Redis.Password,
			PoolSize:     poolSize,
			MinIdleConns: minIdle,
			MaxRetries:   maxRetries,
			DialTimeout:  dialTimeout,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			TLSConfig:    tlsCfg,
		})
		if err := client.Ping(ctx).Err(); err != nil {
			return codeerror.RedisError.Msg("dao.Init redis cluster.Ping error: " + err.Error())
		}
		RDB = client
	} else {
		client := redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			PoolSize:     poolSize,
			MinIdleConns: minIdle,
			MaxRetries:   maxRetries,
			DialTimeout:  dialTimeout,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			TLSConfig:    tlsCfg,
		})
		if err := client.Ping(ctx).Err(); err != nil {
			return codeerror.RedisError.Msg("dao.Init redis.Ping error: " + err.Error())
		}
		RDB = client
	}

	logger.L().InfoCtx(context.Background(), "Redis connected",
		logger.String("addr", addr),
		logger.Int("db", cfg.Redis.DB),
		logger.Int("poolSize", poolSize),
		logger.Int("minIdleConns", minIdle),
		logger.Bool("isCluster", cfg.Redis.IsCluster),
		logger.Bool("isTLS", cfg.Redis.IsTLS),
	)

	return nil
}

// orDefault 返回 val，若 val <= 0 则返回 defaultVal
func orDefault(val, defaultVal int) int {
	if val <= 0 {
		return defaultVal
	}
	return val
}

// buildTLSConfig 构建 TLS 配置，isTLS 为 false 时返回 nil
func buildTLSConfig(isTLS bool) *tls.Config {
	if !isTLS {
		return nil
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
}
