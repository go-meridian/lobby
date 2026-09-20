package redis

import (
	"context"
	"sync"
	"time"

	mq "github.com/SilentQianyi/mq"
	"github.com/SilentQianyi/logger"
	"github.com/redis/go-redis/v9"
)

// redisPubSubSubscription Redis Pub/Sub 订阅实现
type redisPubSubSubscription struct {
	pubsub  *redis.PubSub
	subject string
	handler func(msg mq.Message)
	logger  *logger.Logger
	ctx     context.Context
	running bool
	mu      sync.Mutex
}

func (s *redisPubSubSubscription) start() {
	s.running = true
	go s.listen()
	s.logger.Info("Redis Pub/Sub subscription started", logger.String("subject", s.subject))
}

func (s *redisPubSubSubscription) listen() {
	ch := s.pubsub.Channel()
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Pub/Sub listener stopped", logger.String("subject", s.subject))
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			m := &redisMessage{
				subject:   s.subject,
				data:      []byte(msg.Payload),
				timestamp: time.Now(),
			}
			s.handler(m)
		}
	}
}

func (s *redisPubSubSubscription) Unsubscribe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.running = false
		return s.pubsub.Close()
	}
	return nil
}

func (s *redisPubSubSubscription) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// redisStreamSubscription Redis Streams 订阅实现（无消费者组）
type redisStreamSubscription struct {
	rdb       *redis.Client
	streamKey string
	subject   string
	handler   func(msg mq.Message)
	logger    *logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	mu        sync.Mutex
	lastID    string
}

func (s *redisStreamSubscription) start() {
	ctx, cancel := context.WithCancel(s.ctx)
	s.ctx = ctx
	s.cancel = cancel
	s.running = true
	s.lastID = "0" // 从头开始消费

	go s.listen()
	s.logger.Info("Redis Stream subscription started", logger.String("stream", s.streamKey))
}

func (s *redisStreamSubscription) listen() {
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Stream listener stopped", logger.String("stream", s.streamKey))
			return
		default:
			streams, err := s.rdb.XRead(s.ctx, &redis.XReadArgs{
				Streams: []string{s.streamKey, s.lastID},
				Block:   2 * time.Second,
				Count:   1,
			}).Result()

			if err == redis.Nil {
				continue
			}
			if err != nil {
				if s.ctx.Err() != nil {
					return
				}
				s.logger.Error("XRead error", logger.Error(err))
				continue
			}

			for _, stream := range streams {
				for _, xmsg := range stream.Messages {
					s.lastID = xmsg.ID
					data, _ := xmsg.Values["data"].(string)
					m := &redisMessage{
						subject:   s.subject,
						data:      []byte(data),
						timestamp: time.Now(),
					}
					s.handler(m)
				}
			}
		}
	}
}

func (s *redisStreamSubscription) Unsubscribe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.cancel()
		s.running = false
	}
	return nil
}

func (s *redisStreamSubscription) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// redisQueueSubscription Redis Streams 消费者组订阅实现
type redisQueueSubscription struct {
	rdb       *redis.Client
	streamKey string
	group     string
	consumer  string
	pool      *mq.WorkerPool
	logger    *logger.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	running   bool
	mu        sync.Mutex
}

func (s *redisQueueSubscription) start() {
	ctx, cancel := context.WithCancel(s.ctx)
	s.ctx = ctx
	s.cancel = cancel
	s.running = true

	go s.listen()
	s.logger.Info("Redis queue subscription started",
		logger.String("stream", s.streamKey),
		logger.String("group", s.group),
		logger.String("consumer", s.consumer),
	)
}

func (s *redisQueueSubscription) listen() {
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Queue listener stopped",
				logger.String("stream", s.streamKey),
				logger.String("group", s.group),
			)
			return
		default:
			streams, err := s.rdb.XReadGroup(s.ctx, &redis.XReadGroupArgs{
				Group:    s.group,
				Consumer: s.consumer,
				Streams:  []string{s.streamKey, ">"},
				Block:    2 * time.Second,
				Count:    1,
			}).Result()

			if err == redis.Nil {
				continue
			}
			if err != nil {
				if s.ctx.Err() != nil {
					return
				}
				s.logger.Error("XReadGroup error", logger.Error(err))
				continue
			}

			for _, stream := range streams {
				for _, xmsg := range stream.Messages {
					data, _ := xmsg.Values["data"].(string)
					m := &redisStreamMessage{
						subject:   s.streamKey,
						streamKey: s.streamKey,
						group:     s.group,
						id:        xmsg.ID,
						data:      []byte(data),
						timestamp: time.Now(),
						ackFn: func(id string) error {
							return s.rdb.XAck(s.ctx, s.streamKey, s.group, id).Err()
						},
					}
					s.pool.Submit(m)
				}
			}
		}
	}
}

func (s *redisQueueSubscription) Unsubscribe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.cancel()
		s.pool.Stop()
		s.running = false
	}
	return nil
}

func (s *redisQueueSubscription) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
