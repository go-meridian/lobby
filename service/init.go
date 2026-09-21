package service

import (
	"github.com/go-meridian/logger"
)

// Publisher 发布器接口
type Publisher interface {
	Publish(cmd string, data []byte) error
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
