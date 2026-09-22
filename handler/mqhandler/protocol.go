package mqhandler

import (
	"sync/atomic"

	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/lobby/model/proto/common"
	"github.com/go-meridian/lobby/model/proto/gate"
	"github.com/go-meridian/logger"
	"github.com/go-meridian/mq"
	"google.golang.org/protobuf/proto"
)

// MQHandler MQ 消息处理函数签名（含 msgId 用于路由）
type MQHandler func(connId uint64, requestId uint64, msgId uint32, payload []byte) ([]byte, *codeerror.CodeError)

// natsReqIDCounter NATS 请求 ID 计数器
var natsReqIDCounter uint64

// generateNatsRequestID 生成 NATS 请求 ID
func generateNatsRequestID() uint64 {
	return atomic.AddUint64(&natsReqIDCounter, 1)
}

// handleMessage 解析 Protobuf 消息并路由到对应 handler
func handleMessage(msg mq.Message, routeHandler MQHandler) {
	// 解析 GateRequest protobuf
	var req gate.GateRequest
	if err := proto.Unmarshal(msg.Data(), &req); err != nil {
		log.Error("MQ GateRequest unmarshal error",
			logger.String("subject", msg.Subject()),
			logger.Error(err),
		)
		return
	}

	requestId := req.RequestId
	if requestId == 0 {
		requestId = generateNatsRequestID()
	}

	// 路由到对应 handler
	payload, ce := routeHandler(req.ConnId, requestId, req.MsgId, req.Payload)

	// 构造 GateResponse
	resp := &gate.GateResponse{
		ConnId:    req.ConnId,
		RequestId: requestId,
		MsgId:     req.MsgId,
	}

	if ce != nil {
		// 构造错误的 ErrMsg 放入 payload
		errMsg := &common.ErrMsg{
			RetCode: int32(ce.GetCode()),
			ErrMsg:  ce.Error(),
		}
		errBytes, _ := proto.Marshal(errMsg)
		resp.Payload = errBytes
	} else {
		resp.Payload = payload
	}

	// 回写响应（Request/Reply 模式）
	replyTo := msg.ReplyTo()
	if replyTo == "" {
		return
	}

	respBytes, err := proto.Marshal(resp)
	if err != nil {
		log.Error("MQ GateResponse marshal error",
			logger.Uint64("requestId", requestId),
			logger.Error(err),
		)
		return
	}

	if ce := client.Publish(replyTo, respBytes); ce != nil {
		log.Error("MQ response publish error",
			logger.Uint64("requestId", requestId),
			logger.String("replyTo", replyTo),
			logger.String("error", ce.Error()),
		)
	}
}
