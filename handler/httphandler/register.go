package httphandler

import (
	modelHTTP "github.com/go-meridian/lobby/model/http"
	"github.com/go-meridian/logger"
	"github.com/labstack/echo/v4"
)

// Register 注册 HTTP 路由和中间件
func Register(e *echo.Echo) {
	h := modelHTTP.NewAPIHandler(log)

	// 中间件（按注册顺序执行：Recover → RequestID → RateLimit → AccessLog）
	e.Use(Recover(log))
	e.Use(RequestID)
	e.Use(RateLimit())
	e.Use(AccessLog(log))

	// 路由注册
	e.GET("/health", HandleHealthFunc())
	e.POST("/api/gateway", HandleGateway(h))

	log.Info("HTTP routes registered",
		logger.String("GET", "/health"),
		logger.String("POST", "/api/gateway"),
	)
}
