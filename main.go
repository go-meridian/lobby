package main

import (
	"context"
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
	"github.com/go-meridian/lobby/handler/electhandler"
	"github.com/go-meridian/lobby/handler/eventhandler"
	"github.com/go-meridian/lobby/handler/httphandler"
	"github.com/go-meridian/lobby/handler/mqhandler"
	"github.com/go-meridian/lobby/mq"
	"github.com/go-meridian/lobby/service"
	"github.com/go-meridian/logger"
	mqLib "github.com/go-meridian/mq"
	"github.com/labstack/echo/v4"
)

func main() {
	// ========== 1. 基础设施 ==========
	cfg, ce := config.Init()
	if ce != nil {
		panic("config.Init error: " + ce.Error())
	}

	logCfg := &logger.Config{
		Level:     cfg.Log.Level,
		LogFile:   cfg.Log.LogFile,
		ErrorFile: cfg.Log.ErrorFile,
		LogDir:    cfg.Log.LogDir,
		MaxSize:   cfg.Log.MaxSize,
		MaxAge:    cfg.Log.MaxAge,
	}
	_, err := logger.Init(logCfg)
	if err != nil {
		panic("logger.Init error: " + err.Error())
	}
	defer logger.Close()
	log := logger.L()

	job.Init(nil)
	event.Init(job.Mgr())

	// ========== 2. 存储层 ==========
	if ce := db.Init(cfg); ce != nil {
		log.FatalCtx(context.Background(), "db.Init error", logger.String("error", ce.Error()))
	}
	if ce := dao.Init(cfg); ce != nil {
		log.FatalCtx(context.Background(), "dao.Init error", logger.String("error", ce.Error()))
	}

	mqCfg := &mqLib.Config{
		Mode: mqLib.ModeNATS,
		NATS: &mqLib.NATSConfig{URL: cfg.NATS.URL},
	}
	mqClient, err := mqLib.NewClient(mqCfg)
	if err != nil {
		log.FatalCtx(context.Background(), "mq.NewClient error", logger.Error(err))
	}
	defer mqClient.Close()

	// ========== 3. Handler 层 ==========
	mq.Init(mqClient, log)
	mqhandler.Init(log, cfg.MQ)
	httphandler.Init(log)
	eventhandler.Init(log)
	electhandler.Init(cfg, log)
	defer elect.Close()

	// ========== 4. Service 层 ==========
	service.Init(log, mq.GetPublisher())

	// ========== 5. 注册与启动 ==========
	if ce := mq.Start(); ce != nil {
		log.FatalCtx(context.Background(), "mq.Start error", logger.String("error", ce.Error()))
	}

	mqhandler.Register()
	e := echo.New()
	httphandler.Register(e)
	eventhandler.Register()

	// ========== 6. 运行 ==========
	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	log.InfoCtx(context.Background(), "Lobby server started successfully", logger.String("addr", addr))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.InfoCtx(context.Background(), "HTTP server listening", logger.String("addr", addr))
		if err := e.Start(addr); err != nil {
			log.ErrorCtx(context.Background(), "HTTP server stopped", logger.Error(err))
		}
	}()

	<-quit
	log.InfoCtx(context.Background(), "Shutting down...")

	mq.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.ErrorCtx(context.Background(), "HTTP server shutdown error", logger.Error(err))
	}
	job.Mgr().StopAll(10 * time.Second)
	log.InfoCtx(context.Background(), "Server exited")
}
