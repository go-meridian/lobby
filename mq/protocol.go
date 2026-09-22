package mq

import (
	"sync/atomic"

	"github.com/go-meridian/lobby/model/proto/common"
	"github.com/go-meridian/lobby/model/proto/gate"
	"github.com/go-meridian/logger"
	mqLib "github.com/go-meridian/mq"
	"google.golang.org/protobuf/proto"
)

// natsReqIDCounter NATS 请求 ID 计数器
var natsReqIDCounter uint64

// generateNatsRequestID 生成 NATS 请求 ID
func generateNatsRequestID() uint64 {
	return atomic.AddUint64(&natsReqIDCounter, 1)
}

// handleMessage 解析 Protobuf 消息并路由到对应 handler
func handleMessage(msg mqLib.Message, routeHandler Handler) {
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

	payload, ce := routeHandler(req.ConnId, requestId, req.MsgId, req.Payload)

	resp := &gate.GateResponse{
		ConnId:    req.ConnId,
		RequestId: requestId,
		MsgId:     req.MsgId,
	}

	if ce != nil {
		errMsg := &common.ErrMsg{
			RetCode: int32(ce.GetCode()),
			ErrMsg:  ce.Error(),
		}
		errBytes, _ := proto.Marshal(errMsg)
		resp.Payload = errBytes
	} else {
		resp.Payload = payload
	}

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
