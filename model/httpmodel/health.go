package httpmodel

// HealthRequest 健康检查请求
type HealthRequest struct {
	Verbose bool `query:"verbose"` // 是否返回详细信息
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Checks  map[string]string `json:"checks"`
}
