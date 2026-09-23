package mqhandler

import (
	"context"

	"github.com/go-meridian/lobby/config"
	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/lobby/model/proto/packet"
	"github.com/go-meridian/lobby/mq"
	"github.com/go-meridian/logger"
)

var log *logger.Logger

// MsgHandler 消息处理函数签名（各 service 通过 init 自注册）
type MsgHandler func(ctx context.Context, connId uint64, requestId uint64, payload []byte) ([]byte, *codeerror.CodeError)

var msgRegistry = make(map[uint32]MsgHandler)

// RegisterFn 注册 MsgId 对应的处理函数
func RegisterFn(msgId uint32, fn MsgHandler) {
	msgRegistry[msgId] = fn
}

// RouteMsg 根据 MsgId 分发到对应 handler
func RouteMsg(ctx context.Context, connId uint64, requestId uint64, msgId uint32, payload []byte) ([]byte, *codeerror.CodeError) {
	log.InfoCtx(ctx, "RouteMsg",
		logger.Uint64("connId", connId),
		logger.Uint32("msgId", msgId),
	)

	fn, ok := msgRegistry[msgId]
	if !ok {
		return nil, codeerror.UnknownCmd.Msg("unknown msgId")
	}
	return fn(ctx, connId, requestId, payload)
}

// Init 初始化 mqhandler 层
func Init(l *logger.Logger, mqCfg *config.MQConfig) {
	log = l

	subscribeSubject := "gate2lobby.*"
	publishSubject := "lobby2gate"
	workerCount := 8
	if mqCfg != nil {
		if mqCfg.SubscribeSubject != "" {
			subscribeSubject = mqCfg.SubscribeSubject
		}
		if mqCfg.PublishSubject != "" {
			publishSubject = mqCfg.PublishSubject
		}
		if mqCfg.WorkerCount > 0 {
			workerCount = mqCfg.WorkerCount
		}
	}

	// Gate→Lobby Core NATS 订阅（同步请求-响应）
	mq.RegisterCoreSubscription(subscribeSubject, RouteMsg, workerCount)

	// Lobby→Gate 异步发布
	mq.RegisterPublishStream(publishSubject)
}

func Register() {
	// MsgId 注册
	RegisterFn(uint32(packet.MsgId_MSG_HEARTBEAT_REQ), heartbeatHandler)
}
