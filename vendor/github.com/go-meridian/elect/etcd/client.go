package etcd

import (
	"context"
	"fmt"
	"time"

	"github.com/go-meridian/elect"
	"github.com/go-meridian/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

func init() {
	elect.RegisterFactory(elect.ModeEtcd, NewElector)
}

type etcdElector struct {
	client   *clientv3.Client
	session  *concurrency.Session
	election *concurrency.Election
	log      *logger.Logger

	isLeader bool
	leaderID string
	prefix   string
	ttl      int

	onLeader func()
	onDemote func()

	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func NewElector(cfg *elect.Config, opts *elect.Options) (elect.Elector, error) {
	log := logger.L()
	prefix := elect.OrDefaultStr(cfg.Prefix, "/elect/leader")
	ttl := elect.OrDefault(cfg.TTL, 10)
	dialTimeout := time.Duration(elect.OrDefault(cfg.Etcd.DialTimeout, 5000)) * time.Millisecond

	// 连接 etcd
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Etcd.Endpoints,
		Username:    cfg.Etcd.Username,
		Password:    cfg.Etcd.Password,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("elect etcd clientv3.New error: %w", err)
	}

	// 创建 Session（自动续约）
	session, err := concurrency.NewSession(client,
		concurrency.WithTTL(ttl),
	)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("elect etcd concurrency.NewSession error: %w", err)
	}

	// 创建 Election
	election := concurrency.NewElection(session, prefix)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	e := &etcdElector{
		client:   client,
		session:  session,
		election: election,
		log:      log,
		leaderID: cfg.LeaderID,
		prefix:   prefix,
		ttl:      ttl,
		onLeader: opts.OnLeader,
		onDemote: opts.OnDemote,
		ctx:      ctx,
		cancel:   cancel,
		done:     done,
	}

	// 启动选主协程
	go e.campaign()

	log.Info("elect etcd initialized",
		logger.String("leaderID", cfg.LeaderID),
		logger.String("prefix", prefix),
		logger.Int("ttl", ttl),
		logger.String("endpoints", fmt.Sprintf("%v", cfg.Etcd.Endpoints)),
	)

	return e, nil
}

func (e *etcdElector) IsLeader() bool {
	return e.isLeader
}

func (e *etcdElector) LeaderID() string {
	return e.leaderID
}

func (e *etcdElector) Close() {
	if e.cancel != nil {
		e.cancel()
	}
	if e.done != nil {
		<-e.done
	}
	if e.session != nil {
		e.session.Close()
	}
	if e.client != nil {
		e.client.Close()
	}
	e.log.Info("elect etcd closed", logger.String("leaderID", e.leaderID))
}

func (e *etcdElector) campaign() {
	defer close(e.done)

	for {
		select {
		case <-e.ctx.Done():
			if e.isLeader {
				resignCtx, resignCancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := e.election.Resign(resignCtx); err != nil {
					e.log.Warn("elect etcd Resign error", logger.String("error", err.Error()))
				}
				resignCancel()
				e.isLeader = false
				e.log.Info("elect etcd resigned", logger.String("leaderID", e.leaderID))
			}
			return
		default:
		}

		if err := e.election.Campaign(context.Background(), e.leaderID); err != nil {
			if err == context.Canceled {
				return
			}
			e.log.Error("elect etcd Campaign error",
				logger.String("error", err.Error()),
				logger.String("leaderID", e.leaderID),
			)
			select {
			case <-e.ctx.Done():
				return
			case <-time.After(3 * time.Second):
				continue
			}
		}

		e.isLeader = true
		e.log.Info("elect etcd became leader", logger.String("leaderID", e.leaderID))

		if e.onLeader != nil {
			e.onLeader()
		}

		observeCtx, observeCancel := context.WithCancel(e.ctx)
		go func() {
			defer observeCancel()
			ch := e.election.Observe(observeCtx)
			for resp := range ch {
				if len(resp.Kvs) > 0 {
					currentLeader := string(resp.Kvs[0].Value)
					if currentLeader != e.leaderID {
						e.log.Info("elect etcd leadership changed",
							logger.String("currentLeader", currentLeader),
							logger.String("self", e.leaderID),
						)
						break
					}
				}
			}
		}()

		<-observeCtx.Done()

		if e.ctx.Err() != nil {
			return
		}

		e.isLeader = false
		e.log.Warn("elect etcd lost leadership", logger.String("leaderID", e.leaderID))

		if e.onDemote != nil {
			e.onDemote()
		}
	}
}
