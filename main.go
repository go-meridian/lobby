package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-meridian/elect"
	_ "github.com/go-meridian/elect/etcd"
	_ "github.com/go-meridian/elect/redis"
	"github.com/go-meridian/event"
	"github.com/go-meridian/job"
	"github.com/go-meridian/lobby/config"
	"github.com/go-meridian/lobby/dao"
	"github.com/go-meridian/lobby/db"
	"github.com/go-meridian/lobby/handler"
	electhandler "github.com/go-meridian/lobby/handler/elect"
	_ "github.com/go-meridian/lobby/handler/event"
	httpHandler "github.com/go-meridian/lobby/handler/http"
	natsHandler "github.com/go-meridian/lobby/handler/mq"
	"github.com/go-meridian/lobby/service"
	"github.com/go-meridian/logger"
	"github.com/go-meridian/mq"
	"github.com/labstack/echo/v4"
)

func main() {
	// ========== 1. 基础设施 ==========
	cfg, ce := config.Init()
	if ce != nil {
		panic("config.Init error: " + ce.Error())
	}

	// 初始化 logger
	logCfg := &logger.Config{
		Level:   cfg.Log.Level,
		LogFile: cfg.Log.LogFile,
		LogDir:  "logs",
		MaxSize: cfg.Log.MaxSize,
		MaxAge:  cfg.Log.MaxAge,
	}
	_, err := logger.Init(logCfg)
	if err != nil {
		panic("logger.Init error: " + err.Error())
	}
	defer logger.Close()

	log := logger.L()

	// 初始化 jobmgr
	job.Init(nil)

	// 初始化事件总线
	event.Init(job.Mgr())

	// ========== 2. 存储层 ==========
	if ce := db.Init(cfg); ce != nil {
		log.Fatal("db.Init error", logger.String("error", ce.Error()))
	}

	if ce := dao.Init(cfg); ce != nil {
		log.Fatal("dao.Init error", logger.String("error", ce.Error()))
	}

	// 创建 MQ 客户端
	mqCfg := &mq.Config{
		Mode: mq.ModeNATS,
		NATS: &mq.NATSConfig{URL: cfg.NATS.URL},
	}
	mqClient, err := mq.NewClient(mqCfg)
	if err != nil {
		log.Fatal("mq.NewClient error", logger.Error(err))
	}
	defer mqClient.Close()

	// ========== 2.5 选主 ==========
	electhandler.Init(cfg, log)
	defer elect.Close()

	// ========== 3. Handler 层 ==========
	handler.Init(log)
	httpHandler.Init(log)
	natsHandler.Init(mqClient, log)

	// ========== 4. Service 层 ==========
	service.Init(log, natsHandler.GetPublisher())

	// ========== 5. 注册（init 自注册 + 显式注册） ==========
	// cmd 注册：service/ping.go 等通过 init() 调用 handler.Register 自注册
	// NATS 队列注册：handler/nats/init.go 通过 init() 自注册
	if ce := natsHandler.Register(); ce != nil {
		log.Fatal("natsHandler.Register error", logger.String("error", ce.Error()))
	}

	// ========== 6. 启动 ==========
	e := echo.New()
	httpHandler.Register(e)

	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.Info("Lobby server started successfully", logger.String("addr", addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("HTTP server listening", logger.String("addr", addr))
		if err := e.Start(addr); err != nil {
			log.Info("HTTP server stopped", logger.Error(err))
		}
	}()

	<-quit
	log.Info("Shutting down...")

	// 等待所有 job 完成
	job.Mgr().StopAll(10 * time.Second)
}
