package mq

// Mode MQ 模式
type Mode string

const (
	ModeNATS  Mode = "nats"
	ModeRedis Mode = "redis"
)

// Config MQ 配置
type Config struct {
	Mode  Mode         `json:"mode" yaml:"mode"`
	NATS  *NATSConfig  `json:"nats,omitempty" yaml:"nats,omitempty"`
	Redis *RedisConfig `json:"redis,omitempty" yaml:"redis,omitempty"`
}

// NATSConfig NATS 连接配置
type NATSConfig struct {
	URL string `json:"url" yaml:"url"`
}

// RedisMQMode Redis MQ 子模式
type RedisMQMode string

const (
	RedisMQModeStream RedisMQMode = "streams" // Redis Streams（持久化、消费者组、ACK）
	RedisMQModePubSub RedisMQMode = "pubsub"  // Redis Pub/Sub + List（轻量级）
)

// RedisConfig Redis MQ 配置
type RedisConfig struct {
	Host     string      `json:"host" yaml:"host"`
	Port     int         `json:"port" yaml:"port"`
	Password string      `json:"password" yaml:"password"`
	DB       int         `json:"db" yaml:"db"`
	MQMode   RedisMQMode `json:"mqMode" yaml:"mqMode"` // streams / pubsub
}

// QueueConfig 队列配置
type QueueConfig struct {
	// 通用配置
	StreamName    string `json:"streamName" yaml:"streamName"`
	StreamSubject string `json:"streamSubject" yaml:"streamSubject"`
	ConsumerName  string `json:"consumerName" yaml:"consumerName"`
	WorkerCount   int    `json:"workerCount" yaml:"workerCount"`
	AckWait       int    `json:"ackWait" yaml:"ackWait"` // 秒
	MaxDeliver    int    `json:"maxDeliver" yaml:"maxDeliver"`

	// Redis Streams 特有
	MaxLen int64 `json:"maxLen,omitempty" yaml:"maxLen,omitempty"` // Stream 最大长度
}
