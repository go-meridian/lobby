package httphandler

import (
	"net/http"

	"github.com/go-meridian/lobby/handler"
	"github.com/go-meridian/lobby/model/gateway"
	httpModel "github.com/go-meridian/lobby/model/http"
	"github.com/go-meridian/logger"
	"github.com/labstack/echo/v4"
)

// HandleGateway 处理来自 Gate 的转发请求
func HandleGateway(h *httpModel.APIHandler) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestID, _ := c.Get(ContextKeyRequestID).(string)

		req := &gatewaymodel.Request{}
		if err := c.Bind(req); err != nil {
			return c.JSON(http.StatusBadRequest, &gatewaymodel.Response{
				Code:    -1,
				Message: "invalid request: " + err.Error(),
			})
		}

		h.Logger.Info("HandleGateway",
			logger.String("requestID", requestID),
			logger.String("cmd", req.Cmd),
			logger.Uint64("uid", req.UID),
			logger.Uint64("sessionId", req.SessionID),
		)

		result, ce := handler.RouteCmd(requestID, req.Cmd, req.UID, req.Data)
		if ce != nil {
			h.Logger.Error("HandleGateway route error",
				logger.String("requestID", requestID),
				logger.String("cmd", req.Cmd),
				logger.String("error", ce.Error()),
			)
			return c.JSON(http.StatusOK, &gatewaymodel.Response{
				RequestID: requestID,
				Code:      int(ce.GetCode()),
				Message:   ce.GetMsg(),
			})
		}

		return c.JSON(http.StatusOK, &gatewaymodel.Response{
			RequestID: requestID,
			Code:      0,
			Message:   "success",
			Data:      result,
		})
	}
}
