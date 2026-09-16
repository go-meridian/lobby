package service

import "go.uber.org/zap"

// PingService 健康检查业务逻辑
func PingService(uid uint64) (interface{}, error) {
	logger.Info("PingService", zap.Uint64("uid", uid))
	return map[string]string{"message": "pong from lobby"}, nil
}
