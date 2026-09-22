package mqhandler

import (
	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/lobby/model/proto/common"
	"github.com/go-meridian/lobby/model/proto/packet"
	"github.com/go-meridian/logger"
	"google.golang.org/protobuf/proto"
)

func init() {
	// Gate→Lobby Core NATS 订阅（同步请求-响应）
	RegisterCoreSubscription("gate2lobby.*", RouteMsg, 8)

	// Lobby→Gate 异步发布
	RegisterPublishStream("LOBBY2GATE", "lobby2gate")

	// MsgId 注册
	Register(uint32(packet.MsgId_MSG_HEARTBEAT_REQ), heartbeatService)
}

// heartbeatService 心跳处理
func heartbeatService(connId uint64, requestId uint64, payload []byte) ([]byte, *codeerror.CodeError) {
	log.Info("heartbeatService",
		logger.Uint64("connId", connId),
		logger.Uint64("requestId", requestId),
	)

	rsp := &common.ErrMsg{
		RetCode: 0,
		ErrMsg:  "pong",
	}
	data, err := proto.Marshal(rsp)
	if err != nil {
		return nil, codeerror.SystemError.Msg("marshal error")
	}
	return data, nil
}
