package codeerror

import ce "github.com/SilentQianyi/codeerror"

// 通用错误码
var (
	Success       = ce.New(0, "success")
	HttpSuccess   = ce.New(200, "success")
	SystemError   = ce.New(1001, "system error")
	ConfigError   = ce.New(1002, "config error")
	DBError       = ce.New(1003, "database error")
	RedisError    = ce.New(1004, "redis error")
	LoggerError   = ce.New(1005, "logger error")
	NATSInitError = ce.New(1006, "NATS init error")
	NATSError     = ce.New(1007, "NATS error")
	StreamError   = ce.New(1008, "stream error")
	ConsumerError = ce.New(1009, "consumer error")
	UnknownCmd    = ce.New(1010, "unknown cmd")
)

// 业务错误码范围：2000-9999
// 项目可在自己的 template.go 中扩展业务错误码
