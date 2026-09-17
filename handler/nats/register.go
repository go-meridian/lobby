package nats

import (
	modelNATS "lobby/model/nats"
	"lobby/service"
)

func init() {
	RegisterQueue("gateway", service.RouteCmd, &modelNATS.QueueConfig{
		StreamConfig: modelNATS.StreamConfig{
			StreamName:    "GATEWAY",
			StreamSubject: "gateway.request",
			ConsumerName:  "lobby-gateway",
			AckWait:       30,
			MaxDeliver:    3,
		},
		WorkerCount: 8,
		BatchSize:   16,
	})
}
