package mq

import ce "github.com/SilentQianyi/codeerror"

var (
	MQConnectError   = ce.New(1020, "mq connect error")
	MQPublishError   = ce.New(1021, "mq publish error")
	MQSubscribeError = ce.New(1022, "mq subscribe error")
	MQRequestError   = ce.New(1023, "mq request error")
	MQStreamError    = ce.New(1024, "mq stream error")
	MQConsumerError  = ce.New(1025, "mq consumer error")
	MQTimeoutError   = ce.New(1026, "mq timeout error")
	MQClosedError    = ce.New(1027, "mq client closed")
)
