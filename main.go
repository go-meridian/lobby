package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"lobby/common/logger"
	"lobby/config"
	"lobby/dao"
	"lobby/db"
	httpHandler "lobby/handler/httphandler"
	natsHandler "lobby/handler/nats"
	"lobby/nats"
	"lobby/service"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func main() {
	e := echo.New()

	// 初始化配置
	cfg, ce := config.Init()
	if ce != nil {
		panic("config.Init error: " + ce.Error())
	}

	// 初始化日志
	if ce := logger.Init(cfg); ce != nil {
		panic("logger.Init error: " + ce.Error())
	}
	zapLog := logger.Get()
	defer zapLog.Sync()

	// 初始化 MongoDB
	if ce := db.Init(cfg); ce != nil {
		zapLog.Fatal("db.Init error", zap.String("error", ce.Error()))
	}

	// 初始化 Redis
	if ce := dao.Init(cfg); ce != nil {
		zapLog.Fatal("dao.Init error", zap.String("error", ce.Error()))
	}

	// 初始化 service 层
	service.Init(zapLog)

	// 初始化 HTTP handler 层
	httpHandler.Init(zapLog)

	// 初始化 NATS 连接
	nc, ce := nats.Init(cfg.NATS, zapLog)
	if ce != nil {
		zapLog.Fatal("nats.Init error", zap.String("error", ce.Error()))
	}
	defer nc.Close()

	// 初始化 NATS handler 层
	natsHandler.Init(nc, zapLog)
	if ce := natsHandler.Register(); ce != nil {
		zapLog.Fatal("natsHandler.Register error", zap.String("error", ce.Error()))
	}

	// 注册 HTTP 路由
	httpHandler.Register(e)

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
		zapLog.Info("Lobby HTTP server starting", zap.String("addr", addr))
		if err := e.Start(addr); err != nil {
			zapLog.Info("HTTP server stopped", zap.Error(err))
		}
	}()

	<-quit
	zapLog.Info("Shutting down...")
}
