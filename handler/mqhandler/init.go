package mqhandler

import (
	"github.com/go-meridian/logger"
	"github.com/go-meridian/mq"
)

var (
	client    mq.MQClient
	log       *logger.Logger
	publisher *mqPublisher
)

// Init 初始化 MQ handler 层
func Init(c mq.MQClient, l *logger.Logger) {
	client = c
	log = l
}

// IsConnected 检查 MQ 连接状态
func IsConnected() bool {
	if client == nil {
		return false
	}
	return client.IsConnected()
}
