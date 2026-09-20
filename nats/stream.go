package nats

import (
	"errors"
	"fmt"
	natsModel "lobby/model/nats"
	"time"

	"lobby/model/codeerror"

	natsClient "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// EnsureStream 确保 JetStream Stream 和 Consumer 存在（消费端使用）
func (c *Client) EnsureStream(cfg *natsModel.StreamConfig) *codeerror.CodeError {
	js, err := c.Conn.JetStream()
	if err != nil {
		return codeerror.StreamError.Msg("JetStream context error: " + err.Error())
	}

	streamCfg := &natsClient.StreamConfig{
		Name:      cfg.StreamName,
		Subjects:  []string{fmt.Sprintf("%s.*", cfg.StreamSubject)},
		Storage:   natsClient.FileStorage,
		Retention: natsClient.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
	}

	_, err = js.AddStream(streamCfg)
	if err != nil && !errors.Is(err, natsClient.ErrStreamNameAlreadyInUse) {
		return codeerror.StreamError.Msg("add stream error: " + err.Error())
	}
	c.logger.Info("JetStream stream ensured", zap.String("stream", cfg.StreamName))

	consumerCfg := &natsClient.ConsumerConfig{
		Durable:       cfg.ConsumerName,
		DeliverPolicy: natsClient.DeliverAllPolicy,
		AckPolicy:     natsClient.AckExplicitPolicy,
		AckWait:       time.Duration(cfg.AckWait) * time.Second,
		MaxDeliver:    cfg.MaxDeliver,
		BackOff: []time.Duration{
			1 * time.Second,
			5 * time.Second,
			15 * time.Second,
		},
	}

	_, err = js.AddConsumer(cfg.StreamName, consumerCfg)
	if err != nil && !errors.Is(err, natsClient.ErrConsumerNameAlreadyInUse) {
		return codeerror.ConsumerError.Msg("add consumer error: " + err.Error())
	}
	c.logger.Info("JetStream consumer ensured", zap.String("consumer", cfg.ConsumerName))

	return nil
}

// EnsurePublishStream 确保 JetStream Stream 存在（发送端使用，不创建 Consumer）
func (c *Client) EnsurePublishStream(streamName, subjectPrefix string) *codeerror.CodeError {
	js, err := c.Conn.JetStream()
	if err != nil {
		return codeerror.StreamError.Msg("JetStream context error: " + err.Error())
	}

	streamCfg := &natsClient.StreamConfig{
		Name:      streamName,
		Subjects:  []string{fmt.Sprintf("%s.*", subjectPrefix)},
		Storage:   natsClient.FileStorage,
		Retention: natsClient.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
	}

	_, err = js.AddStream(streamCfg)
	if err != nil && !errors.Is(err, natsClient.ErrStreamNameAlreadyInUse) {
		return codeerror.StreamError.Msg("add stream error: " + err.Error())
	}
	c.logger.Info("JetStream publish stream ensured", zap.String("stream", streamName))

	return nil
}

// JetStreamPublish 发布消息到 JetStream
func (c *Client) JetStreamPublish(subject string, data []byte) *codeerror.CodeError {
	js, err := c.Conn.JetStream()
	if err != nil {
		return codeerror.StreamError.Msg("JetStream context error: " + err.Error())
	}

	_, err = js.Publish(subject, data)
	if err != nil {
		return codeerror.NATSError.Msg("publish error: " + err.Error())
	}
	return nil
}

// JetStream 获取 JetStream context
func (c *Client) JetStream() (natsClient.JetStreamContext, error) {
	return c.Conn.JetStream()
}
