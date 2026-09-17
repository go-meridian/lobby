package httphandler

import (
	"lobby/handler"
	"lobby/model/gateway"
	httpModel "lobby/model/http"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// HandleGateway 处理来自 Gate 的转发请求
func HandleGateway(h *httpModel.APIHandler) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestID, _ := c.Get(ContextKeyRequestID).(string)

		req := &gateway.Request{}
		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, &gateway.Response{
				Code:    -1,
				Message: "invalid request: " + err.Error(),
			})
		}

		h.Logger.Info("HandleGateway",
			zap.String("requestID", requestID),
			zap.String("cmd", req.Cmd),
			zap.Uint64("uid", req.UID),
			zap.Uint64("sessionId", req.SessionID),
		)

		result, ce := handler.RouteCmd(requestID, req.Cmd, req.UID, req.Data)
		if ce != nil {
			h.Logger.Error("HandleGateway route error",
				zap.String("requestID", requestID),
				zap.String("cmd", req.Cmd),
				zap.String("error", ce.Error()),
			)
			return c.JSON(http.StatusOK, &gateway.Response{
				RequestID: requestID,
				Code:      int(ce.GetCode()),
				Message:   ce.GetMsg(),
			})
		}

		return c.JSON(http.StatusOK, &gateway.Response{
			RequestID: requestID,
			Code:      0,
			Message:   "success",
			Data:      result,
		})
	}
}
