package httphandler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-meridian/lobby/dao"
	"github.com/go-meridian/lobby/db"
	"github.com/go-meridian/lobby/handler/mqhandler"
	"github.com/go-meridian/lobby/model/httpmodel"
	"github.com/go-meridian/logger"
	"github.com/labstack/echo/v4"
)

// HandleHealthFunc 健康检查 HTTP 接口
func HandleHealthFunc() echo.HandlerFunc {
	return func(c echo.Context) error {
		// 解析查询参数
		req := &httpmodel.HealthRequest{}
		if err := c.Bind(req); err != nil {
			// 忽略绑定错误，使用默认值
		}

		checks := make(map[string]string)

		// 检查 MongoDB
		checks["mongodb"] = checkMongoDB()

		// 检查 Redis
		checks["redis"] = checkRedis(c.Request().Context())

		// 检查 NATS
		checks["nats"] = checkNATS()

		// 判断整体状态
		status := "ok"
		for _, v := range checks {
			if v != "ok" {
				status = "degraded"
				break
			}
		}

		// 根据 verbose 参数决定是否返回详细信息
		resp := httpmodel.HealthResponse{
			Status:  status,
			Service: "lobby",
		}

		if req.Verbose {
			resp.Checks = checks
		}

		log.Info("Health check response", logger.Any("response", resp))

		if status == "ok" {
			return c.JSON(http.StatusOK, resp)
		}
		return c.JSON(http.StatusServiceUnavailable, resp)
	}
}

// checkMongoDB 检查 MongoDB 连接
func checkMongoDB() string {
	if db.MDB == nil {
		return "not initialized"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.MDB.Client().Ping(ctx, nil); err != nil {
		return "unavailable"
	}
	return "ok"
}

// checkRedis 检查 Redis 连接
func checkRedis(ctx context.Context) string {
	if dao.RDB == nil {
		return "not initialized"
	}
	if err := dao.RDB.Ping(ctx).Err(); err != nil {
		return "unavailable"
	}
	return "ok"
}

// checkNATS 检查 NATS 连接
func checkNATS() string {
	if !mqhandler.IsConnected() {
		return "unavailable"
	}
	return "ok"
}
