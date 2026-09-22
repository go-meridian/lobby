# 架构约定

## HTTP Handler 模式

### 结构体定义

HTTP 请求/响应结构体定义在 `model/httpmodel/` 下，按功能建文件：

```
model/httpmodel/
├── handler.go    # APIHandler 结构体（可选，当前为空）
└── health.go     # HealthRequest / HealthResponse
```

### Handler 实现

处理函数在 `handler/httphandler/` 下实现：

```
handler/httphandler/
├── init.go       # Init() + APIHandler + NewAPIHandler
├── register.go   # Register(e *echo.Echo) 路由注册
├── health.go     # HandleHealthFunc()
├── gateway.go    # HandleGateway(h)
└── middleware.go  # 中间件
```

### Health Check 示例

```go
// model/httpmodel/health.go
type HealthRequest struct {
    Verbose bool `query:"verbose"` // 是否返回详细信息
}

type HealthResponse struct {
    Status  string            `json:"status"`
    Service string            `json:"service"`
    Checks  map[string]string `json:"checks"` // 仅 verbose=true 时返回
}

// handler/httphandler/health.go
func HandleHealthFunc() echo.HandlerFunc {
    return func(c echo.Context) error {
        req := &httpmodel.HealthRequest{}
        if err := c.Bind(req); err != nil { /* 忽略 */ }

        checks := map[string]string{
            "mongodb": checkMongoDB(),
            "redis":   checkRedis(c.Request().Context()),
            "nats":    checkNATS(),
        }

        status := "ok"
        for _, v := range checks {
            if v != "ok" { status = "degraded"; break }
        }

        resp := httpmodel.HealthResponse{Status: status, Service: "lobby"}
        if req.Verbose { resp.Checks = checks }

        if status == "ok" {
            return c.JSON(http.StatusOK, resp)
        }
        return c.JSON(http.StatusServiceUnavailable, resp)
    }
}
```

### 健康检查依赖

| 服务 | 检查方式 |
|------|---------|
| MongoDB | `db.MDB.Client().Ping(ctx, nil)` |
| Redis | `dao.RDB.Ping(ctx).Err()` |
| NATS | `mqhandler.IsConnected()` |

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
// handler/mq/register.go
func init() {
    RegisterCoreSubscription("gate2lobby.*", handler.RouteCmd, 8)
    RegisterPublishStream("LOBBY2GATE", "lobby2gate")
}
```

### HTTP 路由注册

```go
// handler/httphandler/init.go
func Register(e *echo.Echo) {
    e.GET("/health", HandleHealthFunc())
    e.POST("/api/gatewaymodel", HandleGateway(h))
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
2. 在 `init.go` 的 `Register()` 中注册路由

### 新增 NATS 队列

1. `handler/mq/register.go` 的 `init()` 中调用 `RegisterCoreSubscription` 或 `RegisterPublishStream`

## Job 定时任务

使用外部库 `github.com/go-meridian/job` 管理定时任务。

```go
// main.go 初始化
job.Init(nil)
defer job.Mgr().StopAll(10 * time.Second)
event.Init(job.Mgr())  // 事件总线基于 job 管理器
```

- 任务调度由 job 库驱动，项目内无独立 job 目录
- 选主成功后通过回调启动/停止定时任务

## Elect 选主机制

路径：`../../handler/electhandler`

- 封装选主初始化，支持 **etcd** 和 **redis** 两种后端
- 通过 blank import 注册后端：`_ "elect/etcd"` / `_ "elect/redis"`
- 配置项：`config.yaml` 中的 `cfg.Elect`
- 回调机制：
  - `WithOnLeader` — 成为 Leader 时启动定时任务
  - `WithOnDemote` — 失去 Leader 时停止任务

## Event 事件系统

基于 job 管理器的事件总线：

| 层 | 路径 | 职责 |
|---|---|---|
| model/eventmodel/ | 事件结构体定义 | `HealthEvent`、`TopicHealth` 等 |
| handler/event/ | 事件处理函数 | `onHealth(evt)` 等 |

事件结构体实现 `Topic()` 方法，通过事件总线订阅和分发。
