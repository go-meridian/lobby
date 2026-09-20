# 架构约定

## NATS 通信

| 方向 | 模式 | Subject | 说明 |
|------|------|---------|------|
| Gate→Lobby | NATS Core Request/Reply | `gate2lobby.{cmd}` | 同步，Gate 等待回复 |
| Lobby→Gate | NATS Core Publish | `lobby2gate.{cmd}` | 异步，单向通知 |
| Gate→Lobby | HTTP POST | `/api/gateway` | 降级方案 |

### NATSClient 接口

```go
type NATSClient interface {
    EnsureStream(cfg *StreamConfig) *codeerror.CodeError
    JetStream() (natsLib.JetStreamContext, error)
    Subscribe(subject string, handler natsLib.MsgHandler) (*natsLib.Subscription, error)
    Publish(subject string, data []byte) *codeerror.CodeError      // 异步
    PublishSync(subject string, data []byte) *codeerror.CodeError  // 同步
}
```

- `Publish` — 异步发送，立即返回，适用于日志、通知等非关键场景
- `PublishSync` — 发送后等待 `Flush()` 确认，适用于需要确保送达的场景
- `JetStreamPublish` — 发布到 JetStream（在 `nats/stream.go` 中）

## 注册模式

### Service 自注册

```go
// service/ping.go
func init() {
    handler.Register("PING", PingService)
}
func PingService(requestID string, uid uint64, data interface{}) (interface{}, *codeerror.CodeError) { ... }
```

### NATS 队列自注册

```go
// handler/nats/register.go
func init() {
    RegisterCoreSubscription("gate2lobby.*", handler.RouteCmd, 8)
    RegisterPublishStream("LOBBY2GATE", "lobby2gate")
}
```

### HTTP 路由注册

```go
// handler/httphandler/register.go
func Register(e *echo.Echo) {
    e.GET("/health", HandleHealthFunc())
    e.POST("/api/gateway", HandleGateway(h))
}
```

## 中间件

按注册顺序执行：

1. **Recover** - panic 恢复，日志包含 requestID 和 stack
2. **RequestID** - 生成唯一请求 ID，格式 `http_{seq}_{ts}`
3. **RateLimit** - 基于 IP 限流，100 req/s，突发 200
4. **AccessLog** - 请求日志，包含 requestID/method/uri/status/latency

### 请求 ID 链路追踪

```
HTTP: middleware 生成 → echo.Context → HandleGateway → RouteCmd → Service
NATS: Gate 携带 req.RequestID → WorkerPool 提取（空则自动生成 "nats_{seq}_{ts}"）
```

所有日志统一输出 `requestID` 字段，可串联一次请求的完整链路。

## 初始化顺序

```go
// 1. 基础设施
config.Init()
logger.Init(cfg)

// 2. 存储层
db.Init(cfg)
dao.Init(cfg)
nats.Init(cfg.NATS, zapLog)

// 3. Handler 层
handler.Init(zapLog)
httpHandler.Init(zapLog)
natsHandler.Init(nc, zapLog)

// 4. Service 层
service.Init(zapLog, natsHandler.GetPublisher())

// 5. 注册
natsHandler.Register()

// 6. 启动
httpHandler.Register(e)
```

## 新增命令流程

### 新增 Service 命令

1. `service/` 下新建文件，实现处理函数
2. 在 `init()` 中调用 `handler.Register("CMD", YourService)`

### 新增 HTTP 路由

1. `handler/httphandler/` 下实现处理函数
2. 在 `register.go` 的 `Register()` 中注册路由

### 新增 NATS 队列

1. `handler/nats/register.go` 的 `init()` 中调用 `RegisterCoreSubscription` 或 `RegisterPublishStream`
