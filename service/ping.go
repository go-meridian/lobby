package service

import (
	"github.com/go-meridian/lobby/handler"
	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/logger"
)

func init() {
	handler.Register("PING", PingService)
}

// PingService 健康检查业务逻辑
func PingService(requestID string, uid uint64, data interface{}) (interface{}, *codeerror.CodeError) {
	logger.L().Info("PingService", logger.String("requestID", requestID), logger.Uint64("uid", uid))
	return map[string]string{"message": "pong from lobby"}, nil
}
