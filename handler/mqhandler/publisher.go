package mqhandler

import (
	"strconv"

	"github.com/go-meridian/lobby/model/proto/gate"
	"github.com/go-meridian/mq"
	"google.golang.org/protobuf/proto"
)

// mqPublisher MQ 发布器封装
type mqPublisher struct {
	client  mq.MQClient
	subject string
}

// Publish 发送 GatePush 到指定 connId
func (p *mqPublisher) Publish(connId uint64, msgId uint32, payload []byte) error {
	push := &gate.GatePush{
		ConnId:  connId,
		MsgId:   msgId,
		Payload: payload,
	}
	data, err := proto.Marshal(push)
	if err != nil {
		return err
	}

	subject := p.subject + "." + strconv.FormatUint(uint64(msgId), 10)
	ce := p.client.Publish(subject, data)
	if ce != nil {
		return ce
	}
	return nil
}

// PublishBroadcast 广播消息（connId=0）
func (p *mqPublisher) PublishBroadcast(msgId uint32, payload []byte) error {
	return p.Publish(0, msgId, payload)
}

// GetPublisher 获取发布器
func GetPublisher() *mqPublisher {
	return publisher
}
