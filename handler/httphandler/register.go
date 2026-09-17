package httphandler

import (
	modelHTTP "lobby/model/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Register 注册 HTTP 路由和中间件
func Register(e *echo.Echo) {
	h := modelHTTP.NewAPIHandler(logger)

	// 中间件（按注册顺序执行：Recover → RequestID → RateLimit → AccessLog）
	e.Use(Recover(logger))
	e.Use(RequestID)
	e.Use(RateLimit())
	e.Use(AccessLog(logger))

	// 路由注册
	e.GET("/health", HandleHealthFunc())
	e.POST("/api/gateway", HandleGateway(h))

	logger.Info("HTTP routes registered",
		zap.String("GET", "/health"),
		zap.String("POST", "/api/gateway"),
	)
}
