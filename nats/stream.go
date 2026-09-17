package nats

import (
	"errors"
	"fmt"
	codeerror2 "lobby/model/codeerror"
	natsModel "lobby/model/nats"
	"time"

	natsClient "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// EnsureStream 确保 JetStream Stream 和 Consumer 存在
func (c *Client) EnsureStream(cfg *natsModel.StreamConfig) *codeerror2.CodeError {
	js, err := c.Conn.JetStream()
	if err != nil {
		return codeerror2.StreamError.Msg("JetStream context error: " + err.Error())
	}

	// 创建 Stream
	streamCfg := &natsClient.StreamConfig{
		Name:      cfg.StreamName,
		Subjects:  []string{fmt.Sprintf("%s.*", cfg.StreamSubject)},
		Storage:   natsClient.FileStorage,
		Retention: natsClient.WorkQueuePolicy,
		MaxAge:    24 * time.Hour,
	}

	_, err = js.AddStream(streamCfg)
	if err != nil && !errors.Is(err, natsClient.ErrStreamNameAlreadyInUse) {
		return codeerror2.StreamError.Msg("add stream error: " + err.Error())
	}
	c.logger.Info("JetStream stream ensured", zap.String("stream", cfg.StreamName))

	// 创建 Pull Consumer
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
		return codeerror2.ConsumerError.Msg("add consumer error: " + err.Error())
	}
	c.logger.Info("JetStream consumer ensured", zap.String("consumer", cfg.ConsumerName))

	return nil
}

// JetStream 获取 JetStream context
func (c *Client) JetStream() (natsClient.JetStreamContext, error) {
	return c.Conn.JetStream()
}
