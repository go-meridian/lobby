package httphandler

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-meridian/logger"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var reqIDCounter uint64

// RequestID 请求 ID 中间件，为每次请求生成唯一 ID 用于链路追踪
func RequestID(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestID := c.Request().Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("http_%s_%s",
				strconv.FormatUint(atomic.AddUint64(&reqIDCounter, 1), 10),
				strconv.FormatInt(time.Now().UnixMilli()%100000, 10),
			)
		}
		c.Response().Header().Set("X-Request-ID", requestID)
		ctx := logger.WithRequestId(c.Request().Context(), requestID)
		c.SetRequest(c.Request().WithContext(ctx))
		return next(c)
	}
}

// AccessLog 请求日志中间件，包含 requestID
func AccessLog() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			start := time.Now()
			err := next(c)
			stop := time.Now()
			res := c.Response()
			log.InfoCtx(c.Request().Context(), "request",
				logger.String("method", req.Method),
				logger.String("uri", req.RequestURI),
				logger.Int("status", res.Status),
				logger.Int64("latency_ms", stop.Sub(start).Milliseconds()),
			)
			return err
		}
	}
}

// Recover panic 恢复中间件
func Recover() echo.MiddlewareFunc {
	return middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			log.ErrorCtx(c.Request().Context(), "panic recovered",
				logger.Error(err),
				logger.String("stack", string(stack)),
			)
			return err
		},
	})
}

// RateLimit 限流中间件（基于 IP，每秒 100 请求，突发 200）
func RateLimit() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      100,
				Burst:     200,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(429, map[string]string{
				"code":    "429",
				"message": "rate limit exceeded",
			})
		},
	})
}
