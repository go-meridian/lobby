package mqhandler

import (
	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/logger"
)

// MsgHandler 消息处理函数签名（各 service 通过 init 自注册）
type MsgHandler func(connId uint64, requestId uint64, payload []byte) ([]byte, *codeerror.CodeError)

var msgRegistry = make(map[uint32]MsgHandler)

// Register 注册 MsgId 对应的处理函数
func Register(msgId uint32, fn MsgHandler) {
	msgRegistry[msgId] = fn
}

// RouteMsg 根据 MsgId 分发到对应 handler
func RouteMsg(connId uint64, requestId uint64, msgId uint32, payload []byte) ([]byte, *codeerror.CodeError) {
	log.Info("RouteMsg",
		logger.Uint64("connId", connId),
		logger.Uint64("requestId", requestId),
		logger.Uint32("msgId", msgId),
	)

	fn, ok := msgRegistry[msgId]
	if !ok {
		return nil, codeerror.UnknownCmd.Msg("unknown msgId")
	}
	return fn(connId, requestId, payload)
}
