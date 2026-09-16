package service

import (
	"fmt"

	"go.uber.org/zap"
)

var logger *zap.Logger

// Init 初始化 service 层
func Init(log *zap.Logger) {
	logger = log
}

// RouteCmd 根据命令字分发到对应 service
func RouteCmd(cmd string, uid uint64, data interface{}) (interface{}, error) {
	logger.Info("RouteCmd", zap.String("cmd", cmd), zap.Uint64("uid", uid))

	switch cmd {
	case "PING":
		return PingService(uid)
	default:
		return nil, fmt.Errorf("unknown cmd: %s", cmd)
	}
}