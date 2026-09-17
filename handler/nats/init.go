package nats

import (
	"lobby/model"
	"lobby/model/codeerror"
	modelNATS "lobby/model/nats"
	natsClient "lobby/nats"

	"go.uber.org/zap"
)

var (
	client *natsClient.Client
	logger *zap.Logger
)

type queueEntry struct {
	name    string
	handler model.HandlerFunc
	cfg     *modelNATS.QueueConfig
}

// queueRegistry 队列注册表
var queueRegistry []queueEntry

// Init 初始化 NATS handler 层
func Init(nc *natsClient.Client, log *zap.Logger) {
	client = nc
	logger = log
}

// RegisterQueue 注册 NATS 队列（各模块通过 init 自注册）
func RegisterQueue(name string, handler model.HandlerFunc, cfg *modelNATS.QueueConfig) {
	queueRegistry = append(queueRegistry, queueEntry{
		name:    name,
		handler: handler,
		cfg:     cfg,
	})
}

// Register 启动所有已注册的队列
func Register() *codeerror.CodeError {
	if len(queueRegistry) == 0 {
		return nil
	}

	gatewayHandler := modelNATS.NewGatewayHandler(client, logger)

	for _, q := range queueRegistry {
		gatewayHandler.Queue(q.name, q.handler, q.cfg)
	}

	return gatewayHandler.Start()
}
