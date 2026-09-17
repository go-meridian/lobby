package service

import (
	"lobby/model/codeerror"

	"go.uber.org/zap"
)

func init() {
	Register("PING", PingService)
}

// PingService 健康检查业务逻辑
func PingService(requestID string, uid uint64, data interface{}) (interface{}, *codeerror.CodeError) {
	logger.Info("PingService", zap.String("requestID", requestID), zap.Uint64("uid", uid))
	return map[string]string{"message": "pong from lobby"}, nil
}
