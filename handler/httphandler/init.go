package httphandler

import "go.uber.org/zap"

var logger *zap.Logger

// Init 初始化 HTTP handler 层
func Init(log *zap.Logger) {
	logger = log
}
