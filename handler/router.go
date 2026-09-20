package handler

import (
	"lobby/model/codeerror"

	"go.uber.org/zap"
)

var logger *zap.Logger

// Init 初始化 handler 层
func Init(log *zap.Logger) {
	logger = log
}

// ServiceFunc 业务处理函数签名
type ServiceFunc func(requestID string, uid uint64, data interface{}) (interface{}, *codeerror.CodeError)

var cmdRegistry = make(map[string]ServiceFunc)

// Register 注册 cmd 对应的处理函数（各 service 通过 init 自注册）
func Register(cmd string, fn ServiceFunc) {
	cmdRegistry[cmd] = fn
}

// RouteCmd 根据命令字分发到对应 service
func RouteCmd(requestID string, cmd string, uid uint64, data interface{}) (interface{}, *codeerror.CodeError) {
	logger.Info("RouteCmd", zap.String("requestID", requestID), zap.String("cmd", cmd), zap.Uint64("uid", uid))

	fn, ok := cmdRegistry[cmd]
	if !ok {
		return nil, codeerror.UnknownCmd.Msg("unknown cmd: " + cmd)
	}
	return fn(requestID, uid, data)
}
