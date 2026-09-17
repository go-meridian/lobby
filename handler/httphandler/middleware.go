package httphandler

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

var reqIDCounter uint64

const ContextKeyRequestID = "requestID"

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
		c.Set(ContextKeyRequestID, requestID)
		c.Response().Header().Set("X-Request-ID", requestID)
		return next(c)
	}
}

// AccessLog 请求日志中间件，包含 requestID
func AccessLog(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			requestID, _ := c.Get(ContextKeyRequestID).(string)
			start := time.Now()
			err := next(c)
			stop := time.Now()
			res := c.Response()
			log.Info("request",
				zap.String("requestID", requestID),
				zap.String("method", req.Method),
				zap.String("uri", req.RequestURI),
				zap.Int("status", res.Status),
				zap.Int64("latency_ms", stop.Sub(start).Milliseconds()),
			)
			return err
		}
	}
}

// Recover panic 恢复中间件
func Recover(log *zap.Logger) echo.MiddlewareFunc {
	return middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			requestID, _ := c.Get(ContextKeyRequestID).(string)
			log.Error("panic recovered",
				zap.String("requestID", requestID),
				zap.Error(err),
				zap.String("stack", string(stack)),
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
