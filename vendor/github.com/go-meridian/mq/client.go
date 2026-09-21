package mq

import ce "github.com/go-meridian/codeerror"

// MQClient 统一 MQ 客户端接口
type MQClient interface {
	// Publish 异步发布消息（不等待回复）
	Publish(subject string, data []byte) *ce.CodeError

	// Request 同步请求-等待回复（超时返回错误）
	Request(subject string, data []byte, timeoutMs int) ([]byte, *ce.CodeError)

	// Subscribe 订阅主题（Core 模式，收到消息直接回调）
	Subscribe(subject string, handler func(msg Message)) (Subscription, *ce.CodeError)

	// SubscribeQueue 订阅队列（支持消费者组负载均衡，消息需要 Ack）
	// NATS: JetStream PullSubscribe
	// Redis: XREADGROUP Consumer Group 或 List + BRPOP
	SubscribeQueue(cfg *QueueConfig, handler func(msg Message)) (Subscription, *ce.CodeError)

	// EnsureQueue 确保队列/Stream 存在
	EnsureQueue(cfg *QueueConfig) *ce.CodeError

	// IsConnected 检查连接状态
	IsConnected() bool

	// Close 关闭连接
	Close()
}

// Subscription 订阅句柄
type Subscription interface {
	// Unsubscribe 取消订阅
	Unsubscribe() error

	// IsActive 是否活跃
	IsActive() bool
}
