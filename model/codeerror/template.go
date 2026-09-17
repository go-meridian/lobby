package codeerror

var (
	Success       = New(0, "success")
	HttpSuccess   = New(200, "success")
	SystemError   = New(1001, "system error")
	ConfigError   = New(1002, "config error")
	DBError       = New(1003, "database error")
	RedisError    = New(1004, "redis error")
	LoggerError   = New(1005, "logger error")
	NATSInitError = New(1006, "NATS init error")
	NATSError     = New(1007, "NATS error")
	StreamError   = New(1008, "stream error")
	ConsumerError = New(1009, "consumer error")
	UnknownCmd    = New(1010, "unknown cmd")
)
