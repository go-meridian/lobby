package natsModel

import (
	"lobby/model"
	"lobby/model/codeerror"

	natsLib "github.com/nats-io/nats.go"
)

// NATSClient NATS 客户端接口，用于解耦 nats 包
type NATSClient interface {
	EnsureStream(cfg *StreamConfig) *codeerror.CodeError
	JetStream() (natsLib.JetStreamContext, error)
	Subscribe(subject string, handler natsLib.MsgHandler) (*natsLib.Subscription, error)
	PublishSync(subject string, data []byte) *codeerror.CodeError
}

// StreamConfig Stream 配置
type StreamConfig struct {
	StreamName    string `json:"streamName" yaml:"streamName"`
	StreamSubject string `json:"streamSubject" yaml:"streamSubject"`
	ConsumerName  string `json:"consumerName" yaml:"consumerName"`
	AckWait       int    `json:"ackWait" yaml:"ackWait"`
	MaxDeliver    int    `json:"maxDeliver" yaml:"maxDeliver"`
}

// QueueConfig 队列注册配置（内嵌 StreamConfig）
type QueueConfig struct {
	StreamConfig `yaml:",inline"`
	WorkerCount  int `json:"workerCount" yaml:"workerCount"`
	BatchSize    int `json:"batchSize" yaml:"batchSize"`
}

// pendingQueue 暂存的队列注册信息
type pendingQueue struct {
	name    string
	handler model.HandlerFunc
	cfg     *QueueConfig
}

// subscription 单个队列的订阅实例
type subscription struct {
	name       string
	sub        *natsLib.Subscription
	workerPool *WorkerPool
}
