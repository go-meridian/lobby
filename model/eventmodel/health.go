package eventmodel

// Topic 常量
const (
	TopicHealth = "lobby:health"
)

// HealthEvent 健康检查事件
type HealthEvent struct {
	RequestID string
	Status    string
}

func (e *HealthEvent) Topic() string {
	return TopicHealth
}
