package service

import (
	codeerror2 "lobby/model/codeerror"

	"go.uber.org/zap"
)

// ServiceFunc 业务处理函数签名（cmd 由路由层传入，service 不需要关心）
type ServiceFunc func(requestID string, uid uint64, data interface{}) (interface{}, *codeerror2.CodeError)

var cmdRegistry = make(map[string]ServiceFunc)

// Register 注册 cmd 对应的处理函数（各 service 通过 init 自注册）
func Register(cmd string, fn ServiceFunc) {
	cmdRegistry[cmd] = fn
}

// RouteCmd 根据命令字分发到对应 service
func RouteCmd(requestID string, cmd string, uid uint64, data interface{}) (interface{}, *codeerror2.CodeError) {
	logger.Info("RouteCmd", zap.String("requestID", requestID), zap.String("cmd", cmd), zap.Uint64("uid", uid))

	fn, ok := cmdRegistry[cmd]
	if !ok {
		return nil, codeerror2.UnknownCmd.Msg("unknown cmd: " + cmd)
	}
	return fn(requestID, uid, data)
}
