package dao

import (
	"context"
	"time"

	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/redis/go-redis/v9"
)

// DBData 所有缓存模型需实现此接口
type DBData interface {
	RedisKey() string
	Pack() ([]byte, *codeerror.CodeError)
	UnPack([]byte) *codeerror.CodeError
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
