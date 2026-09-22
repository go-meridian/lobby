package mqhandler

import (
	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/logger"
	"github.com/go-meridian/mq"
)

// coreSubscriptionEntry Core NATS 订阅注册
type coreSubscriptionEntry struct {
	subject     string
	handler     MQHandler
	workerCount int
}

// coreSubscriptionRegistry Core NATS 订阅注册表
var coreSubscriptionRegistry []coreSubscriptionEntry

// publishStreamEntry 发送端 Stream 注册
type publishStreamEntry struct {
	streamName    string
	streamSubject string
}

// publishStreamRegistry 发送端 Stream 注册表
var publishStreamRegistry []publishStreamEntry

// RegisterCoreSubscription 注册 MQ Core 订阅
func RegisterCoreSubscription(subject string, h MQHandler, workerCount int) {
	coreSubscriptionRegistry = append(coreSubscriptionRegistry, coreSubscriptionEntry{
		subject:     subject,
		handler:     h,
		workerCount: workerCount,
	})
}

// RegisterPublishStream 注册 MQ 发送 Stream
func RegisterPublishStream(streamName, streamSubject string) {
	publishStreamRegistry = append(publishStreamRegistry, publishStreamEntry{
		streamName:    streamName,
		streamSubject: streamSubject,
	})
}

// GetRegisteredSubscriptions 获取已注册的订阅信息（用于启动日志）
func GetRegisteredSubscriptions() []string {
	var result []string
	for _, entry := range coreSubscriptionRegistry {
		result = append(result, entry.subject)
	}
	for _, entry := range publishStreamRegistry {
		result = append(result, entry.streamSubject+".{msgId} (publish)")
	}
	return result
}

// Start 启动所有已注册的订阅和发送 Stream
func Start() *codeerror.CodeError {
	for _, entry := range coreSubscriptionRegistry {
		localEntry := entry
		_, ce := client.Subscribe(localEntry.subject, func(msg mq.Message) {
			handleMessage(msg, localEntry.handler)
		})
		if ce != nil {
			return codeerror.SystemError.Msg("MQ subscribe error: " + ce.Error())
		}
		log.Info("MQ subscription registered",
			logger.String("subject", localEntry.subject),
			logger.Int("workers", localEntry.workerCount),
		)
	}

	if len(publishStreamRegistry) > 0 {
		ps := publishStreamRegistry[0]
		publisher = &mqPublisher{
			client:  client,
			subject: ps.streamSubject,
		}
		log.Info("MQ publish stream registered",
			logger.String("stream", ps.streamName),
			logger.String("subject", ps.streamSubject+".{msgId}"),
		)
	}

	return nil
}
