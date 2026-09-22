package mq

import (
	"strconv"

	"github.com/go-meridian/lobby/model/proto/gate"
	mqLib "github.com/go-meridian/mq"
	"google.golang.org/protobuf/proto"
)

// Publisher MQ 发布器
type Publisher struct {
	client  mqLib.MQClient
	subject string
}

// Publish 发送 GatePush 到指定 connId
func (p *Publisher) Publish(connId uint64, msgId uint32, payload []byte) error {
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
func (p *Publisher) PublishBroadcast(msgId uint32, payload []byte) error {
	return p.Publish(0, msgId, payload)
}

// GetPublisher 获取发布器
func GetPublisher() *Publisher {
	return publisher
}
