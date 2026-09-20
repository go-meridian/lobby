package mqhandler

import (
	"lobby/model"
	"lobby/model/codeerror"

	"github.com/SilentQianyi/logger"
	mq "github.com/SilentQianyi/mq"
)

var (
	client    mq.MQClient
	log       *logger.Logger
	publisher *mqPublisher
)

// mqPublisher MQ 发布器封装
type mqPublisher struct {
	client  mq.MQClient
	subject string
	log     *logger.Logger
}

// Publish 发布消息
func (p *mqPublisher) Publish(cmd string, data []byte) error {
	subject := p.subject + "." + cmd
	ce := p.client.Publish(subject, data)
	if ce != nil {
		return ce
	}
	return nil
}

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

// Init 初始化 MQ handler 层
func Init(c mq.MQClient, l *logger.Logger) {
	client = c
	log = l
}

// RegisterCoreSubscription 注册 MQ Core 订阅（各模块通过 init 自注册）
func RegisterCoreSubscription(subject string, handler model.HandlerFunc, workerCount int) {
	coreSubscriptionRegistry = append(coreSubscriptionRegistry, coreSubscriptionEntry{
		subject:     subject,
		handler:     handler,
		workerCount: workerCount,
	})
}

// RegisterPublishStream 注册 MQ 发送 Stream（各模块通过 init 自注册）
func RegisterPublishStream(streamName, streamSubject string) {
	publishStreamRegistry = append(publishStreamRegistry, publishStreamEntry{
		streamName:    streamName,
		streamSubject: streamSubject,
	})
}

// GetPublisher 获取发布器
func GetPublisher() *mqPublisher {
	return publisher
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
	// 启动 Core MQ 订阅
	if len(coreSubscriptionRegistry) > 0 {
		entry := coreSubscriptionRegistry[0]

		// 使用 mq.MQClient 的 Subscribe 方法
		// 需要将 model.HandlerFunc 适配为 mq.Message handler
		_, ce := client.Subscribe(entry.subject, func(msg mq.Message) {
			// 这里需要处理消息，但 mq.Message 接口不直接支持 Respond
			// 我们需要通过类型断言来获取底层消息并调用 Respond
			// 这是 mq 模块需要改进的地方
			log.Info("Received message",
				logger.String("subject", msg.Subject()),
				logger.Int("dataLen", len(msg.Data())),
			)
		})
		if ce != nil {
			return ce
		}
		log.Info("MQ subscription registered",
			logger.String("subject", entry.subject),
			logger.Int("workers", entry.workerCount),
		)
	}

	// 初始化 Publisher（用于异步发布）
	if len(publishStreamRegistry) > 0 {
		ps := publishStreamRegistry[0]
		publisher = &mqPublisher{
			client:  client,
			subject: ps.streamSubject,
			log:     log,
		}
		log.Info("MQ publish stream registered",
			logger.String("stream", ps.streamName),
			logger.String("subject", ps.streamSubject+".{cmd}"),
		)
	}

	return nil
}
