package http

import (
	"net/http"

	"Lobby/service"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// APIHandler API 处理器
type APIHandler struct {
	logger *zap.Logger
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(logger *zap.Logger) *APIHandler {
	return &APIHandler{logger: logger}
}

// GatewayRequest 从 Gate 转发来的请求体
type GatewayRequest struct {
	SessionID uint64      `json:"sessionId"`
	UID       uint64      `json:"uid"`
	Cmd       string      `json:"cmd"`
	Data      interface{} `json:"data"`
}

// GatewayResponse 返回给 Gate 的响应体
type GatewayResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// HandleGateway 处理来自 Gate 的转发请求
func (h *APIHandler) HandleGateway(c echo.Context) error {
	req := &GatewayRequest{}
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, &GatewayResponse{
			Code:    -1,
			Message: "invalid request: " + err.Error(),
		})
	}

	h.logger.Info("HandleGateway",
		zap.String("cmd", req.Cmd),
		zap.Uint64("uid", req.UID),
		zap.Uint64("sessionId", req.SessionID),
	)

	// 根据 Cmd 分发到 service 层
	rsp, err := service.RouteCmd(req.Cmd, req.UID, req.Data)
	if err != nil {
		h.logger.Error("HandleGateway route error", zap.String("cmd", req.Cmd), zap.Error(err))
		return c.JSON(http.StatusOK, &GatewayResponse{
			Code:    -1,
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, rsp)
}
