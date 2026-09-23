package logger

import "context"

// requestIdKey 用于在 context 中存储 requestId，非导出类型避免键冲突
type requestIdKey struct{}

// requestIdFieldKey 是 requestId 附加到日志时的字段名
const requestIdFieldKey = "requestId"

// WithRequestId 将 requestId 写入 context，供带 Ctx 的日志方法自动附加到每条日志
func WithRequestId(ctx context.Context, requestId string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIdKey{}, requestId)
}

// RequestIdFromContext 从 context 读取 requestId，不存在时返回空字符串
func RequestIdFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIdKey{}).(string); ok {
		return id
	}
	return ""
}

// appendRequestId 若 context 中存在非空 requestId，则将其追加到字段末尾
func appendRequestId(ctx context.Context, fields []Field) []Field {
	if ctx == nil {
		return fields
	}
	if id, ok := ctx.Value(requestIdKey{}).(string); ok && id != "" {
		return append(fields, String(requestIdFieldKey, id))
	}
	return fields
}
