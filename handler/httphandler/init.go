package httphandler

import "github.com/go-meridian/logger"

var log *logger.Logger

// Init 初始化 HTTP handler 层
func Init(l *logger.Logger) {
	log = l
}
