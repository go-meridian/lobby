package mq

import (
	"github.com/go-meridian/logger"
	mqLib "github.com/go-meridian/mq"
)

var (
	client    mqLib.MQClient
	log       *logger.Logger
	publisher *Publisher
)

// Init 初始化 MQ 层
func Init(c mqLib.MQClient, l *logger.Logger) {
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
