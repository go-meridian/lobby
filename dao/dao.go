package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"Lobby/config"
)

// RDB Redis 客户端实例
var RDB *redis.Client

// DBData 所有缓存模型需实现此接口
type DBData interface {
	RedisKey() string
	Pack() ([]byte, error)
	UnPack([]byte) error
}

// Init 初始化 Redis
func Init(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := RDB.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("dao.Init redis.Ping error: %w", err)
	}

	return nil
}

// FillDBInfo 读穿透：查 Redis -> 未命中返回 ObjectNotExist
func FillDBInfo(ctx context.Context, key string, data DBData) error {
	val, err := RDB.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return fmt.Errorf("ObjectNotExist")
	}
	if err != nil {
		return err
	}
	return data.UnPack(val)
}

// SetDBInfo 写缓存（TTL 2 小时）
func SetDBInfo(ctx context.Context, data DBData) error {
	val, err := data.Pack()
	if err != nil {
		return err
	}
	return RDB.Set(ctx, data.RedisKey(), val, 2*time.Hour).Err()
}

// DelDBInfo 失效缓存
func DelDBInfo(ctx context.Context, key string) error {
	return RDB.Del(ctx, key).Err()
}
