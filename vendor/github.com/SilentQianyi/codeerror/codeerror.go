package codeerror

// CodeError 带错误码的错误类型
type CodeError struct {
	code int32
	msg  string
}

// New 创建新的 CodeError
func New(code int32, msg string) *CodeError {
	return &CodeError{
		code: code,
		msg:  msg,
	}
}

// GetCode 获取错误码
func (e *CodeError) GetCode() int32 {
	if e == nil {
		return 0
	}
	return e.code
}

// SetCode 设置错误码
func (e *CodeError) SetCode(code int32) {
	e.code = code
}

// GetMsg 获取错误消息文本
func (e *CodeError) GetMsg() string {
	if e == nil {
		return ""
	}
	return e.msg
}

// SetMsg 设置错误消息文本
func (e *CodeError) SetMsg(msg string) {
	e.msg = msg
}

// Msg 返回一个新的 CodeError，继承 code 但替换 msg（不修改原对象）
func (e *CodeError) Msg(msg string) *CodeError {
	return &CodeError{
		code: e.code,
		msg:  msg,
	}
}

// Error 实现 error 接口
func (e *CodeError) Error() string {
	if e == nil {
		return ""
	}
	return e.msg
}
