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
	"lobby/handler"
	httpHandler "lobby/handler/httphandler"
	natsHandler "lobby/handler/nats"
	"lobby/nats"
	"lobby/service"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func main() {
	// ========== 1. 基础设施 ==========
	cfg, ce := config.Init()
	if ce != nil {
		panic("config.Init error: " + ce.Error())
	}

	if ce := logger.Init(cfg); ce != nil {
		panic("logger.Init error: " + ce.Error())
	}
	zapLog := logger.Get()
	defer zapLog.Sync()

	// ========== 2. 存储层 ==========
	if ce := db.Init(cfg); ce != nil {
		zapLog.Fatal("db.Init error", zap.String("error", ce.Error()))
	}

	if ce := dao.Init(cfg); ce != nil {
		zapLog.Fatal("dao.Init error", zap.String("error", ce.Error()))
	}

	nc, ce := nats.Init(cfg.NATS, zapLog)
	if ce != nil {
		zapLog.Fatal("nats.Init error", zap.String("error", ce.Error()))
	}
	defer nc.Close()

	// ========== 3. Handler 层 ==========
	handler.Init(zapLog)
	httpHandler.Init(zapLog)
	natsHandler.Init(nc, zapLog)

	// ========== 4. Service 层 ==========
	service.Init(zapLog, natsHandler.GetPublisher())

	// ========== 5. 注册（init 自注册 + 显式注册） ==========
	// cmd 注册：service/ping.go 等通过 init() 调用 handler.Register 自注册
	// NATS 队列注册：handler/nats/register.go 通过 init() 自注册
	if ce := natsHandler.Register(); ce != nil {
		zapLog.Fatal("natsHandler.Register error", zap.String("error", ce.Error()))
	}

	// ========== 6. 启动 ==========
	e := echo.New()
	httpHandler.Register(e)

	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	zapLog.Info("Lobby server started successfully", zap.String("addr", addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		zapLog.Info("HTTP server listening", zap.String("addr", addr))
		if err := e.Start(addr); err != nil {
			zapLog.Info("HTTP server stopped", zap.Error(err))
		}
	}()

	<-quit
	zapLog.Info("Shutting down...")
}
