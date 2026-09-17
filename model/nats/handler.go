package natsModel

import (
	"context"
	"fmt"
	"lobby/model"
	codeerror2 "lobby/model/codeerror"

	natsLib "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// GatewayHandler NATS 网关处理器，管理多个队列订阅
type GatewayHandler struct {
	client        NATSClient
	subscriptions []*subscription
	pending       []pendingQueue
	logger        *zap.Logger
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewGatewayHandler 创建 NATS 网关处理器
func NewGatewayHandler(client NATSClient, logger *zap.Logger) *GatewayHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &GatewayHandler{
		client: client,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Queue 注册单个队列订阅
func (h *GatewayHandler) Queue(name string, handler model.HandlerFunc, cfg *QueueConfig) {
	h.pending = append(h.pending, pendingQueue{
		name:    name,
		handler: handler,
		cfg:     cfg,
	})
}

// Start 启动所有已注册队列的订阅和 WorkerPool
func (h *GatewayHandler) Start() *codeerror2.CodeError {
	for _, pq := range h.pending {
		if ce := h.addSubscription(pq.name, pq.handler, pq.cfg); ce != nil {
			return ce
		}
	}
	h.pending = nil
	return nil
}

// addSubscription 为单个队列创建订阅
func (h *GatewayHandler) addSubscription(name string, handler model.HandlerFunc, cfg *QueueConfig) *codeerror2.CodeError {
	streamCfg := &cfg.StreamConfig
	if ce := h.client.EnsureStream(streamCfg); ce != nil {
		return ce
	}

	js, err := h.client.JetStream()
	if err != nil {
		return codeerror2.StreamError.Msg("JetStream context error: " + err.Error())
	}

	sub, err := js.PullSubscribe(
		fmt.Sprintf("%s.*", cfg.StreamSubject),
		cfg.ConsumerName,
	)
	if err != nil {
		return codeerror2.ConsumerError.Msg("PullSubscribe error: " + err.Error())
	}

	pool := NewWorkerPool(cfg.WorkerCount, handler, h.logger)
	pool.Start()

	s := &subscription{
		name:       name,
		sub:        sub,
		workerPool: pool,
	}
	h.subscriptions = append(h.subscriptions, s)

	go h.dispatcher(name, sub, cfg.BatchSize, pool)

	h.logger.Info("NATS subscription started",
		zap.String("queue", name),
		zap.String("stream", cfg.StreamName),
		zap.String("consumer", cfg.ConsumerName),
		zap.Int("workers", cfg.WorkerCount),
	)
	return nil
}

// dispatcher 消息分发协程，每个队列独立运行
func (h *GatewayHandler) dispatcher(queueName string, sub *natsLib.Subscription, batchSize int, pool *WorkerPool) {
	for {
		select {
		case <-h.ctx.Done():
			h.logger.Info("Dispatcher stopped", zap.String("queue", queueName))
			return
		default:
			msgs, err := sub.Fetch(batchSize, natsLib.MaxWait(100))
			if err != nil {
				if err == natsLib.ErrTimeout {
					continue
				}
				if h.ctx.Err() != nil {
					return
				}
				h.logger.Error("Fetch failed", zap.String("queue", queueName), zap.Error(err))
				continue
			}

			for _, msg := range msgs {
				pool.Submit(msg)
			}
		}
	}
}

// Stop 停止所有订阅和线程池
func (h *GatewayHandler) Stop() {
	h.cancel()
	for _, s := range h.subscriptions {
		s.workerPool.Stop()
		h.logger.Info("NATS subscription stopped", zap.String("queue", s.name))
	}
}
