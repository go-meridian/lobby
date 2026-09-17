package httpModel

import "go.uber.org/zap"

// APIHandler HTTP API 处理器
type APIHandler struct {
	Logger *zap.Logger
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(logger *zap.Logger) *APIHandler {
	return &APIHandler{Logger: logger}
}
