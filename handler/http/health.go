package http

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HandleHealthFunc 健康检查 HTTP 接口
func HandleHealthFunc() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
		})
	}
}
