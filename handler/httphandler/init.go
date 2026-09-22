package httphandler

import "github.com/go-meridian/logger"

var log *logger.Logger

// Init 初始化 HTTP handler 层
func Init(l *logger.Logger) {
	log = l
}

// APIHandler HTTP API 处理器
type APIHandler struct {
	Logger *logger.Logger
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(log *logger.Logger) *APIHandler {
	return &APIHandler{Logger: log}
}
