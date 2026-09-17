package service

import (
	modelNATS "lobby/model/nats"

	"go.uber.org/zap"
)

var logger *zap.Logger

var publisher *modelNATS.NATSPublisher

// Init 初始化 service 层
func Init(log *zap.Logger, pub *modelNATS.NATSPublisher) {
	logger = log
	publisher = pub
}

// GetPublisher 获取 NATS 发布器
func GetPublisher() *modelNATS.NATSPublisher {
	return publisher
}
