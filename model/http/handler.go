package httpModel

import "github.com/go-meridian/logger"

// APIHandler HTTP API 处理器
type APIHandler struct {
	Logger *logger.Logger
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(log *logger.Logger) *APIHandler {
	return &APIHandler{Logger: log}
}
