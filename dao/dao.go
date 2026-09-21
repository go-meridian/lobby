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

// DBData 所有缓存模型需实现此接口
type DBData interface {
	RedisKey() string
	Pack() ([]byte, *codeerror.CodeError)
	UnPack([]byte) *codeerror.CodeError
}

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

	logger.L().Info("Redis connected",
		logger.String("addr", addr),
		logger.Int("db", cfg.Redis.DB),
		logger.Int("poolSize", poolSize),
		logger.Int("minIdleConns", minIdle),
		logger.Bool("isCluster", cfg.Redis.IsCluster),
		logger.Bool("isTLS", cfg.Redis.IsTLS),
	)

	return nil
}

// FillDBInfo 读穿透：查 Redis -> 未命中返回 ObjectNotExist
func FillDBInfo(ctx context.Context, key string, data DBData) *codeerror.CodeError {
	val, err := RDB.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return codeerror.RedisError.Msg("ObjectNotExist")
	}
	if err != nil {
		return codeerror.RedisError.Msg("FillDBInfo redis.Get error: " + err.Error())
	}
	return data.UnPack(val)
}

// SetDBInfo 写缓存（TTL 2 小时）
func SetDBInfo(ctx context.Context, data DBData) *codeerror.CodeError {
	val, ce := data.Pack()
	if ce != nil {
		return ce
	}
	if err := RDB.Set(ctx, data.RedisKey(), val, 2*time.Hour).Err(); err != nil {
		return codeerror.RedisError.Msg("SetDBInfo redis.Set error: " + err.Error())
	}
	return nil
}

// DelDBInfo 失效缓存
func DelDBInfo(ctx context.Context, key string) *codeerror.CodeError {
	if err := RDB.Del(ctx, key).Err(); err != nil {
		return codeerror.RedisError.Msg("DelDBInfo redis.Del error: " + err.Error())
	}
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
