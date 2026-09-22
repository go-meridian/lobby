package mqhandler

import "github.com/go-meridian/lobby/handler"

func init() {
	// Gate→Lobby Core NATS 订阅（同步请求-响应）
	RegisterCoreSubscription("gate2lobby.*", handler.RouteCmd, 8)

	// Lobby→Gate 异步发布
	RegisterPublishStream("LOBBY2GATE", "lobby2gate")
}
