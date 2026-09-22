package service

import (
	"github.com/go-meridian/logger"
)

// Publisher 发布器接口
type Publisher interface {
	// Publish 发送 GatePush 到指定 connId
	Publish(connId uint64, msgId uint32, payload []byte) error
	// PublishBroadcast 广播消息（connId=0）
	PublishBroadcast(msgId uint32, payload []byte) error
}

var log *logger.Logger

var publisher Publisher

// Init 初始化 service 层
func Init(l *logger.Logger, pub Publisher) {
	log = l
	publisher = pub
}

// GetPublisher 获取发布器
func GetPublisher() Publisher {
	return publisher
}
