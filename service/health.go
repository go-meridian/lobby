package service

import (
	"context"
	"time"

	"github.com/go-meridian/lobby/dao"
	"github.com/go-meridian/lobby/db"
	"github.com/go-meridian/lobby/handler/mqhandler"
)

func checkMongoDB() string {
	if db.MDB == nil {
		return "not initialized"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.MDB.Client().Ping(ctx, nil); err != nil {
		return "unavailable"
	}
	return "ok"
}

func checkRedis() string {
	if dao.RDB == nil {
		return "not initialized"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := dao.RDB.Ping(ctx).Err(); err != nil {
		return "unavailable"
	}
	return "ok"
}

func checkNATS() string {
	if !mqhandler.IsConnected() {
		return "unavailable"
	}
	return "ok"
}
