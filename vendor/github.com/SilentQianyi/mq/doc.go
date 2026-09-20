// Package mq 提供统一的消息队列客户端接口，支持 NATS 和 Redis 两种实现。
//
// 使用示例：
//
//	// 导入实现包以注册工厂（重要！）
//	import (
//	    _ "github.com/SilentQianyi/mq/nats"
//	    _ "github.com/SilentQianyi/mq/redis"
//	)
//
//	// 创建客户端
//	cfg := &mq.Config{
//	    Mode: mq.ModeNATS,
//	    NATS: &mq.NATSConfig{URL: "nats://localhost:4222"},
//	}
//	client, err := mq.NewClient(cfg, logger)
//
//	// 发布消息
//	client.Publish("subject", data)
//
//	// 请求-回复
//	resp, err := client.Request("subject", data, 5000)
//
//	// 订阅主题
//	sub, _ := client.Subscribe("subject", func(msg mq.Message) {
//	    fmt.Println(string(msg.Data()))
//	})
//	defer sub.Unsubscribe()
//
//	// 订阅队列（消费者组）
//	queueSub, _ := client.SubscribeQueue(&mq.QueueConfig{
//	    StreamName:   "mystream",
//	    ConsumerName: "mygroup",
//	    WorkerCount:  4,
//	}, func(msg mq.Message) {
//	    // 处理消息
//	    msg.Ack()
//	})
//	defer queueSub.Unsubscribe()
package mq
