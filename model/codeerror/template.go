package codeerror

import ce "github.com/SilentQianyi/codeerror"

// CodeError re-export，保持对外部包的透明
type CodeError = ce.CodeError

// 通用错误码
var (
	Success     = ce.New(0, "success")
	HttpSuccess = ce.New(200, "success")
	SystemError = ce.New(1001, "system error")
	ConfigError = ce.New(1002, "config error")
	DBError     = ce.New(1003, "database error")
	RedisError  = ce.New(1004, "redis error")
	UnknownCmd  = ce.New(1010, "unknown cmd")
)

// 业务错误码范围：2000-9999
// 项目可在自己的 template.go 中扩展业务错误码
