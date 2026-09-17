package model

import "lobby/model/codeerror"

// HandlerFunc 命令处理函数签名
type HandlerFunc func(requestID string, cmd string, uid uint64, data interface{}) (interface{}, *codeerror.CodeError)
