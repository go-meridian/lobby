package gatewaymodel

// Request Gate 转发给 Lobby 的请求体
type Request struct {
	RequestID string      `json:"requestId"`
	SessionID uint64      `json:"sessionId"`
	UID       uint64      `json:"uid"`
	Cmd       string      `json:"cmd"`
	Data      interface{} `json:"data"`
}

// Response Lobby 返回给 Gate 的响应体
type Response struct {
	RequestID string      `json:"requestId"`
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
}
