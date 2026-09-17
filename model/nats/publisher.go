package natsModel

import (
	"encoding/json"
	"lobby/model/codeerror"
	"lobby/model/gateway"

	"go.uber.org/zap"
)

// PublishFn 发布函数签名
type PublishFn func(subject string, data []byte) *codeerror.CodeError

// NATSPublisher NATS 消息发布器（Lobby→Gate，异步通知场景）
type NATSPublisher struct {
	streamSubject string
	publishFn     PublishFn
	logger        *zap.Logger
}

// NewNATSPublisher 创建发布器
func NewNATSPublisher(streamSubject string, publishFn PublishFn, logger *zap.Logger) *NATSPublisher {
	return &NATSPublisher{
		streamSubject: streamSubject,
		publishFn:     publishFn,
		logger:        logger,
	}
}

// PublishResponse 向 Gate 发送响应消息（异步，不等待回复）
func (p *NATSPublisher) PublishResponse(requestID string, cmd string, rsp *gateway.Response) *codeerror.CodeError {
	subject := p.streamSubject + "." + cmd
	data, err := json.Marshal(rsp)
	if err != nil {
		p.logger.Error("Marshal response failed",
			zap.String("requestID", requestID),
			zap.String("cmd", cmd),
			zap.Error(err),
		)
		return codeerror.NATSError.Msg("marshal response failed: " + err.Error())
	}

	if ce := p.publishFn(subject, data); ce != nil {
		p.logger.Error("Publish response failed",
			zap.String("requestID", requestID),
			zap.String("subject", subject),
			zap.String("error", ce.Error()),
		)
		return ce
	}

	p.logger.Info("Response published (async)",
		zap.String("requestID", requestID),
		zap.String("subject", subject),
	)
	return nil
}
