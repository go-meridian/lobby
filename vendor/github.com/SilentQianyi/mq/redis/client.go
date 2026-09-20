package redis

import (
	"context"
	"fmt"

	mq "github.com/SilentQianyi/mq"
	"github.com/SilentQianyi/logger"
	ce "github.com/SilentQianyi/codeerror"
	"github.com/redis/go-redis/v9"
)

func init() {
	mq.RegisterFactory(mq.ModeRedis, func(cfg *mq.Config) (mq.MQClient, error) {
		if cfg.Redis == nil {
			return nil, fmt.Errorf("redis config is required")
		}
		return NewRedisClient(cfg.Redis)
	})
}

// redisClient Redis 客户端实现
type redisClient struct {
	rdb    *redis.Client
	mqMode mq.RedisMQMode
	logger *logger.Logger
	ctx    context.Context
}

// NewRedisClient 创建 Redis 客户端
func NewRedisClient(cfg *mq.RedisConfig) (mq.MQClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connect error: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log := mq.GetLogger()
	log.Info("Redis connected",
		logger.String("addr", addr),
		logger.Int("db", cfg.DB),
		logger.String("mqMode", string(cfg.MQMode)),
	)

	return &redisClient{
		rdb:    rdb,
		mqMode: cfg.MQMode,
		logger: log,
		ctx:    ctx,
	}, nil
}

// Publish 发布消息
func (c *redisClient) Publish(subject string, data []byte) *ce.CodeError {
	switch c.mqMode {
	case mq.RedisMQModePubSub:
		return c.publishPubSub(subject, data)
	case mq.RedisMQModeStream:
		return c.publishStream(subject, data)
	default:
		return c.publishStream(subject, data)
	}
}

// publishPubSub 使用 Pub/Sub 发布（无持久化）
func (c *redisClient) publishPubSub(subject string, data []byte) *ce.CodeError {
	channel := "mq:pubsub:" + subject
	if err := c.rdb.Publish(c.ctx, channel, data).Err(); err != nil {
		return mq.MQPublishError.Msg("redis publish: " + err.Error())
	}
	return nil
}

// publishStream 使用 Streams 发布（有持久化）
func (c *redisClient) publishStream(subject string, data []byte) *ce.CodeError {
	streamKey := "mq:stream:" + subject
	args := &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"data": data,
		},
	}
	if err := c.rdb.XAdd(c.ctx, args).Err(); err != nil {
		return mq.MQPublishError.Msg("redis xadd: " + err.Error())
	}
	return nil
}

// Request 同步请求-等待回复（使用 List + BRPOP）
func (c *redisClient) Request(subject string, data []byte, timeoutMs int) ([]byte, *ce.CodeError) {
	// 1. 生成唯一回复标识
	replyID := generateID()
	replyKey := "mq:reply:" + replyID

	// 2. 构造请求消息，附带 replyTo
	envelope := &requestEnvelope{
		Data:    data,
		ReplyTo: replyKey,
	}
	payload, err := jsonMarshal(envelope)
	if err != nil {
		return nil, mq.MQRequestError.Msg("marshal envelope: " + err.Error())
	}

	// 3. LPUSH 到请求队列
	reqKey := "mq:req:" + subject
	if err := c.rdb.LPush(c.ctx, reqKey, payload).Err(); err != nil {
		return nil, mq.MQRequestError.Msg("redis lpush: " + err.Error())
	}

	// 4. BRPOP 等待回复
	timeout := fmt.Sprintf("%dms", timeoutMs)
	result, err := c.rdb.BRPop(c.ctx, parseDuration(timeout), replyKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, mq.MQTimeoutError.Msg("redis brpop timeout")
		}
		return nil, mq.MQRequestError.Msg("redis brpop: " + err.Error())
	}

	// result[0] 是 key, result[1] 是 value
	return []byte(result[1]), nil
}

// Subscribe 订阅主题
func (c *redisClient) Subscribe(subject string, handler func(msg mq.Message)) (mq.Subscription, *ce.CodeError) {
	switch c.mqMode {
	case mq.RedisMQModePubSub:
		return c.subscribePubSub(subject, handler)
	case mq.RedisMQModeStream:
		return c.subscribeStream(subject, handler)
	default:
		return c.subscribeStream(subject, handler)
	}
}

// subscribePubSub 使用 Pub/Sub 订阅
func (c *redisClient) subscribePubSub(subject string, handler func(msg mq.Message)) (mq.Subscription, *ce.CodeError) {
	channel := "mq:pubsub:" + subject
	pubsub := c.rdb.Subscribe(c.ctx, channel)

	// 等待订阅确认
	if _, err := pubsub.Receive(c.ctx); err != nil {
		return nil, mq.MQSubscribeError.Msg("redis subscribe: " + err.Error())
	}

	sub := &redisPubSubSubscription{
		pubsub:  pubsub,
		subject: subject,
		handler: handler,
		logger:  c.logger,
		ctx:     c.ctx,
	}
	sub.start()

	return sub, nil
}

// subscribeStream 使用 Streams 订阅
func (c *redisClient) subscribeStream(subject string, handler func(msg mq.Message)) (mq.Subscription, *ce.CodeError) {
	streamKey := "mq:stream:" + subject

	sub := &redisStreamSubscription{
		rdb:       c.rdb,
		streamKey: streamKey,
		subject:   subject,
		handler:   handler,
		logger:    c.logger,
		ctx:       c.ctx,
	}
	sub.start()

	return sub, nil
}

// SubscribeQueue 订阅队列（支持消费者组负载均衡）
func (c *redisClient) SubscribeQueue(cfg *mq.QueueConfig, handler func(msg mq.Message)) (mq.Subscription, *ce.CodeError) {
	// 确保队列存在
	if ce := c.EnsureQueue(cfg); ce != nil {
		return nil, ce
	}

	streamKey := "mq:stream:" + cfg.StreamName
	group := cfg.ConsumerName
	consumer := fmt.Sprintf("%s-%s", group, generateID()[:8])

	pool := mq.NewWorkerPool(cfg.WorkerCount, handler)
	pool.Start()

	sub := &redisQueueSubscription{
		rdb:       c.rdb,
		streamKey: streamKey,
		group:     group,
		consumer:  consumer,
		pool:      pool,
		logger:    c.logger,
		ctx:       c.ctx,
	}
	sub.start()

	return sub, nil
}

// EnsureQueue 确保 Redis Stream 和 Consumer Group 存在
func (c *redisClient) EnsureQueue(cfg *mq.QueueConfig) *ce.CodeError {
	streamKey := "mq:stream:" + cfg.StreamName
	group := cfg.ConsumerName

	// XGROUP CREATE streamKey group 0 MKSTREAM
	err := c.rdb.XGroupCreateMkStream(c.ctx, streamKey, group, "0").Err()
	if err != nil && !isBusyGroupErr(err) {
		return mq.MQStreamError.Msg("xgroup create: " + err.Error())
	}
	c.logger.Info("Redis stream group ensured",
		logger.String("stream", streamKey),
		logger.String("group", group),
	)

	return nil
}

// IsConnected 检查连接状态
func (c *redisClient) IsConnected() bool {
	return c.rdb.Ping(c.ctx).Err() == nil
}

// Close 关闭连接
func (c *redisClient) Close() {
	if c.rdb != nil {
		c.rdb.Close()
		c.logger.Info("Redis connection closed")
	}
}
