package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"Lobby/config"
)

var (
	DB  *gorm.DB
	RDB *redis.Client
)

// DBData 所有缓存模型需实现此接口
type DBData interface {
	RedisKey() string
	Pack() ([]byte, error)
	UnPack([]byte) error
}

// Init 初始化 MySQL + Redis
func Init(cfg *config.Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Database)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("dao.Init gorm.Open error: %w", err)
	}

	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
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

// TableByUid 按 UID 分表（每 200 万一个分片）
func TableByUid(uid uint64, table string) string {
	shard := uid / 2000000
	return fmt.Sprintf("%s_%d", table, shard)
}
