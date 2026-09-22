package httphandler

import (
	"github.com/go-meridian/logger"
	"github.com/labstack/echo/v4"
)

var log *logger.Logger

// Init 初始化 httphandler 层
func Init(l *logger.Logger) {
	log = l
}

// Register 注册 HTTP 路由和中间件
func Register(e *echo.Echo) {

	// 中间件（按注册顺序执行：Recover → RequestID → RateLimit → AccessLog）
	e.Use(Recover())
	e.Use(RequestID)
	e.Use(RateLimit())
	e.Use(AccessLog())

	// 路由注册
	e.GET("/health", HandleHealthFunc())

	log.Info("HTTP routes registered",
		logger.String("GET", "/health"),
		logger.String("POST", "/api/proto"),
	)
}
