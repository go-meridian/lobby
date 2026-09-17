package nats

import (
	"lobby/model"
	"lobby/model/codeerror"
	modelNATS "lobby/model/nats"
	natsClient "lobby/nats"

	"go.uber.org/zap"
)

var (
	client         *natsClient.Client
	logger         *zap.Logger
	publisher      *modelNATS.NATSPublisher
	coreSubscriber *modelNATS.CoreSubscriber
)

// coreSubscriptionEntry Core NATS 订阅注册
type coreSubscriptionEntry struct {
	subject     string
	handler     model.HandlerFunc
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

// Init 初始化 NATS handler 层
func Init(nc *natsClient.Client, log *zap.Logger) {
	client = nc
	logger = log
}

// RegisterCoreSubscription 注册 NATS Core 订阅（各模块通过 init 自注册）
func RegisterCoreSubscription(subject string, handler model.HandlerFunc, workerCount int) {
	coreSubscriptionRegistry = append(coreSubscriptionRegistry, coreSubscriptionEntry{
		subject:     subject,
		handler:     handler,
		workerCount: workerCount,
	})
}

// RegisterPublishStream 注册 NATS 发送 Stream（各模块通过 init 自注册）
func RegisterPublishStream(streamName, streamSubject string) {
	publishStreamRegistry = append(publishStreamRegistry, publishStreamEntry{
		streamName:    streamName,
		streamSubject: streamSubject,
	})
}

// GetPublisher 获取发布器
func GetPublisher() *modelNATS.NATSPublisher {
	return publisher
}

// GetCoreSubscriber 获取 Core 订阅器
func GetCoreSubscriber() *modelNATS.CoreSubscriber {
	return coreSubscriber
}

// GetRegisteredSubscriptions 获取已注册的订阅信息（用于启动日志）
func GetRegisteredSubscriptions() []string {
	var result []string
	for _, entry := range coreSubscriptionRegistry {
		result = append(result, entry.subject)
	}
	for _, entry := range publishStreamRegistry {
		result = append(result, entry.streamSubject+".{cmd} (publish)")
	}
	return result
}

// Register 启动所有已注册的订阅和发送 Stream
func Register() *codeerror.CodeError {
	// 启动 Core NATS 订阅
	if len(coreSubscriptionRegistry) > 0 {
		entry := coreSubscriptionRegistry[0]
		coreSubscriber = modelNATS.NewCoreSubscriber(client, entry.subject, entry.handler, entry.workerCount, logger)
		if err := coreSubscriber.Start(); err != nil {
			return codeerror.NATSError.Msg("Core subscriber start error: " + err.Error())
		}
		logger.Info("NATS subscription registered",
			zap.String("subject", entry.subject),
			zap.Int("workers", entry.workerCount),
		)
	}

	// 初始化 Publisher（用于异步发布）
	if len(publishStreamRegistry) > 0 {
		ps := publishStreamRegistry[0]
		publisher = modelNATS.NewNATSPublisher(ps.streamSubject, client.PublishSync, logger)
		logger.Info("NATS publish stream registered",
			zap.String("stream", ps.streamName),
			zap.String("subject", ps.streamSubject+".{cmd}"),
		)
	}

	return nil
}
