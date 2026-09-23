package mq

import (
	"context"

	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/logger"
	mqLib "github.com/go-meridian/mq"
)

// Handler MQ 消息处理函数签名（含 msgId 用于路由）
type Handler func(ctx context.Context, connId uint64, requestId uint64, msgId uint32, payload []byte) ([]byte, *codeerror.CodeError)

// coreSubscriptionEntry Core NATS 订阅注册
type coreSubscriptionEntry struct {
	subject     string
	handler     Handler
	workerCount int
}

// coreSubscriptionRegistry Core NATS 订阅注册表
var coreSubscriptionRegistry []coreSubscriptionEntry

// activeSubscription 运行时已激活的订阅状态
type activeSubscription struct {
	sub  mqLib.Subscription
	pool *mqLib.WorkerPool
}

// activeSubscriptions 运行时已激活的订阅（用于优雅关闭）
var activeSubscriptions []activeSubscription

// publishStreamRegistry 发送端 Stream 主题注册表
var publishStreamRegistry []string

// RegisterCoreSubscription 注册 MQ Core 订阅
func RegisterCoreSubscription(subject string, h Handler, workerCount int) {
	coreSubscriptionRegistry = append(coreSubscriptionRegistry, coreSubscriptionEntry{
		subject:     subject,
		handler:     h,
		workerCount: workerCount,
	})
}

// RegisterPublishStream 注册 MQ 发送 Stream 主题
func RegisterPublishStream(streamSubject string) {
	publishStreamRegistry = append(publishStreamRegistry, streamSubject)
}

// GetRegisteredSubscriptions 获取已注册的订阅信息（用于启动日志）
func GetRegisteredSubscriptions() []string {
	var result []string
	for _, entry := range coreSubscriptionRegistry {
		result = append(result, entry.subject)
	}
	for _, subject := range publishStreamRegistry {
		result = append(result, subject+".{msgId} (publish)")
	}
	return result
}

// Start 启动所有已注册的订阅和发送 Stream
func Start() *codeerror.CodeError {
	for i := range coreSubscriptionRegistry {
		entry := &coreSubscriptionRegistry[i]

		pool := mqLib.NewWorkerPool(entry.workerCount, func(msg mqLib.Message) {
			handleMessage(msg, entry.handler)
		})
		pool.Start()

		sub, ce := client.Subscribe(entry.subject, pool.Submit)
		if ce != nil {
			pool.Stop()
			err := codeerror.SystemError.Msg("MQ subscribe error: " + ce.Error())
			log.ErrorCtx(context.Background(), "MQ subscription failed",
				logger.String("subject", entry.subject),
				logger.String("error", err.Error()),
			)
			return err
		}

		activeSubscriptions = append(activeSubscriptions, activeSubscription{sub: sub, pool: pool})
		log.InfoCtx(context.Background(), "MQ subscription registered",
			logger.String("subject", entry.subject),
			logger.Int("workers", entry.workerCount),
		)
	}

	if len(publishStreamRegistry) > 0 {
		subject := publishStreamRegistry[0]
		publisher = &Publisher{
			client:  client,
			subject: subject,
		}
		log.InfoCtx(context.Background(), "MQ publish stream registered",
			logger.String("subject", subject+".{msgId}"),
		)
	}

	return nil
}

// Stop 停止所有订阅和 worker pool（优雅关闭）
func Stop() {
	for i := range activeSubscriptions {
		entry := &activeSubscriptions[i]
		if entry.sub != nil {
			if err := entry.sub.Unsubscribe(); err != nil {
				log.ErrorCtx(context.Background(), "MQ unsubscribe error", logger.String("error", err.Error()))
			}
		}
		if entry.pool != nil {
			entry.pool.Stop()
		}
	}
	activeSubscriptions = nil
}
