package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/go-meridian/elect"
	"github.com/go-meridian/logger"
	"github.com/redis/go-redis/v9"
)

func init() {
	elect.RegisterFactory(elect.ModeRedis, NewElector)
}

type redisElector struct {
	rdb      redis.Cmdable
	log      *logger.Logger
	lockKey  string
	leaderID string
	ttl      int

	isLeader bool

	onLeader func()
	onDemote func()

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	leaderCtx    context.Context
	leaderCancel context.CancelFunc
}

func NewElector(cfg *elect.Config, opts *elect.Options) (elect.Elector, error) {
	log := logger.L()
	lockKey := elect.OrDefaultStr(cfg.Prefix, "/elect/leader")
	ttl := elect.OrDefault(cfg.TTL, 10)

	// 创建 redis 客户端
	addr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("elect redis connect error: %w", err)
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	done := make(chan struct{})

	e := &redisElector{
		rdb:      rdb,
		log:      log,
		lockKey:  lockKey,
		leaderID: cfg.LeaderID,
		ttl:      ttl,
		onLeader: opts.OnLeader,
		onDemote: opts.OnDemote,
		ctx:      ctx2,
		cancel:   cancel2,
		done:     done,
	}

	// 启动选主协程
	go e.campaign()

	log.Info("elect redis initialized",
		logger.String("leaderID", cfg.LeaderID),
		logger.String("lockKey", lockKey),
		logger.Int("ttl", ttl),
		logger.String("addr", addr),
	)

	return e, nil
}

func (e *redisElector) IsLeader() bool {
	return e.isLeader
}

func (e *redisElector) LeaderID() string {
	return e.leaderID
}

func (e *redisElector) Close() {
	if e.cancel != nil {
		e.cancel()
	}
	if e.done != nil {
		<-e.done
	}
	if e.rdb != nil {
		if client, ok := e.rdb.(*redis.Client); ok {
			client.Close()
		}
	}
	e.log.Info("elect redis closed", logger.String("leaderID", e.leaderID))
}

func (e *redisElector) campaign() {
	defer close(e.done)

	for {
		select {
		case <-e.ctx.Done():
			if e.isLeader {
				// 释放锁
				e.rdb.Del(context.Background(), e.lockKey)
				e.isLeader = false
				e.log.Info("elect redis released lock", logger.String("leaderID", e.leaderID))
			}
			return
		default:
		}

		// 尝试获取锁
		ok, err := e.rdb.SetNX(e.ctx, e.lockKey, e.leaderID, time.Duration(e.ttl)*time.Second).Result()
		if err != nil {
			if e.ctx.Err() != nil {
				return
			}
			e.log.Error("elect redis SetNX error", logger.String("error", err.Error()))
			select {
			case <-e.ctx.Done():
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}

		if ok {
			// 成为 Leader
			e.isLeader = true
			e.leaderCtx, e.leaderCancel = context.WithCancel(e.ctx)
			e.log.Info("elect redis became leader", logger.String("leaderID", e.leaderID))

			if e.onLeader != nil {
				e.onLeader()
			}

			// 启动续租协程
			go e.renew()

			// 等待失去 Leader
			<-e.leaderCtx.Done()

			e.isLeader = false
			e.log.Warn("elect redis lost leadership", logger.String("leaderID", e.leaderID))

			if e.onDemote != nil {
				e.onDemote()
			}
		} else {
			// 成为 Follower，定期重试
			select {
			case <-e.ctx.Done():
				return
			case <-time.After(1 * time.Second):
			}
		}
	}
}

func (e *redisElector) renew() {
	renewInterval := time.Duration(e.ttl/3) * time.Second
	if renewInterval < 1*time.Second {
		renewInterval = 1 * time.Second
	}
	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.leaderCtx.Done():
			return
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			// 续租
			ok, err := e.rdb.Expire(e.ctx, e.lockKey, time.Duration(e.ttl)*time.Second).Result()
			if err != nil {
				e.log.Error("elect redis Expire error", logger.String("error", err.Error()))
				e.leaderCancel()
				return
			}
			if !ok {
				// key 不存在，失去 Leader
				e.log.Warn("elect redis Expire failed, key not found")
				e.leaderCancel()
				return
			}
		}
	}
}
