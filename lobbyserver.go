package main

import (
	"Lobby/common/logger"
	"Lobby/config"
	"Lobby/dao"
	"Lobby/db"
	lhttp "Lobby/handler/http"
	"Lobby/service"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func main() {
	e := echo.New()

	// 初始化配置
	cfg, err := config.Init(e.Logger)
	if err != nil {
		e.Logger.Panicf("main config.Init error! err[ %s ]", err.Error())
	}

	// 初始化日志
	logger, err := logger.InitLogger(cfg, e.Logger)
	if err != nil {
		e.Logger.Panicf("main common.InitLogger error! err[ %s ]", err.Error())
	}
	defer logger.Sync()

	// 初始化 MongoDB
	if err := db.Init(cfg); err != nil {
		logger.Fatal("main db.Init error", zap.Error(err))
	}

	// 初始化 Redis
	if err := dao.Init(cfg); err != nil {
		logger.Fatal("main dao.Init error", zap.Error(err))
	}

	// 初始化 service 层
	service.Init(logger)

	// 初始化 HTTP 处理器
	httpHandler := lhttp.NewAPIHandler(logger)

	// 请求日志中间件
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			start := time.Now()
			err := next(c)
			stop := time.Now()
			res := c.Response()
			logger.Info("request",
				zap.String("method", req.Method),
				zap.String("uri", req.RequestURI),
				zap.Int("status", res.Status),
				zap.Int64("bytes_in", req.ContentLength),
				zap.Int64("bytes_out", res.Size),
				zap.Duration("latency", stop.Sub(start)),
			)
			return err
		}
	})

	// 路由注册
	e.GET("/health", lhttp.HandleHealthFunc())
	e.POST("/api/gateway", httpHandler.HandleGateway)

	addr := fmt.Sprintf(":%d", cfg.Server.HTTPPort)
	logger.Info("Lobby HTTP server starting", zap.String("addr", addr))
	e.Logger.Fatal(e.Start(addr))
}
