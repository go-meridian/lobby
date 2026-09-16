// codeError.go 带错误码的错误类型
// 提供 Code 和 Msg 的封装，支持错误码模板模式

package codeError

// ========== 类型定义 ==========

// CodeError 带错误码的错误类型
type CodeError struct {
	code int32
	msg  string
}

// ========== 构造函数 ==========

// New 创建新的 CodeError
func New(code int32, msg string) *CodeError {
	return &CodeError{
		code: code,
		msg:  msg,
	}
}

// ========== 属性访问 ==========

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
