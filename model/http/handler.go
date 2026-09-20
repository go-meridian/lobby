package httpModel

import "github.com/SilentQianyi/logger"

// APIHandler HTTP API 处理器
type APIHandler struct {
	Logger *logger.Logger
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(log *logger.Logger) *APIHandler {
	return &APIHandler{Logger: log}
}
