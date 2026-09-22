# Lobby 架构文档

## 1. 项目概述

Lobby 是游戏大厅业务服务，与 Gate（WebSocket 网关）组成完整服务架构。Gate 负责 WebSocket 连接管理和会话维护，Lobby 负责所有业务逻辑和数据读写。

**技术栈：** Go + Echo v4 + MongoDB v2 + go-redis v9 + NATS JetStream + zap 日志 + viper 配置

**服务关系：**
```
客户端 --[WebSocket]--> Gate:8080 --[HTTP POST JSON]--> Lobby:9001
                                  --[NATS JetStream]--> Lobby
```

## 2. 目录结构

```
Lobby/
├── main.go                        # 程序入口，编排初始化顺序和生命周期
├── config.yaml                    # 运行时配置
├── config/
│   └── config.go                  # 配置结构体 + viper 加载
├── common/
│   ├── codeError/
│   │   ├── codeError.go           # CodeError 类型（实现 error 接口）
│   │   └── template.go            # 预定义错误码模板
│   └── logger/
│       ├── log.go                 # zap 日志初始化
│       └── logWriter.go           # 按日期+大小轮转的日志写入器
├── db/
│   └── db.go                      # MongoDB 连接初始化（全局 MDB）
├── dao/
│   └── dao.go                     # Redis 缓存层（DBData 接口 + CRUD）
├── model/
│   ├── gateway/
│   │   ├── message.go             # Gate <-> Lobby 请求/响应结构体
│   │   └── subject.go             # NATS Subject 生成工具
│   ├── nats/
│   │   └── config.go              # StreamConfig 结构体
│   └── db/
│       └── redis.go               # 旧版 Redis 模型（未使用）
├── handler/
│   ├── http/
│   │   ├── gateway.go             # POST /api/gateway 处理器
│   │   └── health.go              # GET /health 健康检查
│   ├── nats/
│   │   ├── gateway.go             # NATS 多队列订阅管理器
│   │   └── worker.go              # WorkerPool 消息处理协程池
│   └── c2s/                       # [预留] 客户端直连 MQ
├── nats/
│   ├── nats.go                    # NATS 连接封装
│   └── stream.go                  # JetStream Stream/Consumer 管理
└── service/
    ├── handler.go                 # HandlerFunc 类型 + 处理器注册表
    ├── router.go                  # Init + RouteCmd 命令路由
    └── ping.go                    # PingService 示例
```

## 3. 架构总览

```
                    ┌─────────────────────────────────────┐
                    │           Lobby Service              │
                    │                                      │
  Gate ──HTTP POST──┤──> handler/http/gateway.go ──┐       │
  (WebSocket 网关)   │    Bind -> RouteCmd          │       │
                    │                              v       │
  NATS ──JetStream──┤──> handler/nats/gateway.go   service │
  (消息队列)         │    dispatcher -> WorkerPool──>RouteCmd│
                    │                              │       │
                    │              +-----------+    v       │
                    │              | PingService|  ...      │
                    │              +-----------+    │       │
                    │                     ┌────────┘       │
                    │                     v                │
                    │              ┌──────┴──────┐         │
                    │              │ dao (Redis)  │         │
                    │              │ db  (MongoDB)│         │
                    └──────────────┴─────────────┴─────────┘
```

两条入口（HTTP / NATS）统一汇聚到 `service.RouteCmd`，实现一次编写、双通道复用。

## 4. 分层详解

### 4.1 Handler 层

**HTTP Handler** (`handler/http/gateway.go`)
- 接收 Gate 转发的 `GatewayRequest{SessionID, UID, Cmd, Data}`
- 调用 `service.RouteCmd(cmd, uid, data)` 路由
- 返回 `GatewayResponse` JSON

**NATS Handler** (`handler/nats/gateway.go`)
- `GatewayHandler` 管理多个队列订阅
- 每个队列独立的 `subscription + WorkerPool + dispatcher`
- `WorkerPool.processMsg` 解析消息后调用注册的 `HandlerFunc`

关键类型：
```go
// handler/nats/worker.go
type WorkerPool struct {
    workerCount int
    jobChan     chan *nats.Msg
    handler     service.HandlerFunc  // 每个队列绑定不同的处理函数
    logger      *zap.Logger
    wg          sync.WaitGroup
}

// handler/nats/gatewaymodel.go
type subscription struct {
    name       string
    sub        *natsLib.Subscription
    workerPool *WorkerPool
}
```

### 4.2 Service 层

**处理器注册表** (`service/handler.go`)
```go
type HandlerFunc func(cmd string, uid uint64, data interface{}) (interface{}, *codeError.CodeError)

func RegisterHandler(name string, fn HandlerFunc)  // 注册
func GetHandler(name string) (HandlerFunc, bool)    // 获取
```

**命令路由** (`service/router.go`)
```go
func RouteCmd(cmd string, uid uint64, data interface{}) (interface{}, *codeError.CodeError)
```
按 `cmd` 分发到对应业务函数（如 `PingService`）。

### 4.3 数据访问层

**Redis 缓存** (`dao/dao.go`)
```go
type DBData interface {
    RedisKey() string
    Pack() ([]byte, *codeError.CodeError)
    UnPack([]byte) *codeError.CodeError
}
func FillDBInfo(ctx, key, data DBData) *codeError.CodeError  // 读穿透
func SetDBInfo(ctx, data DBData) *codeError.CodeError        // 写缓存（TTL 2h）
func DelDBInfo(ctx, key) *codeError.CodeError                // 失效缓存
```

**MongoDB** (`db/db.go`)
- 全局 `MDB *mongo.Database`
- 当前仅初始化连接，业务 Collection 操作待扩展

### 4.4 配置层 (`config/config.go`)

```go
type Config struct {
    Server    *ServerConfig
    Log       *LogConfig
    Mongo     *MongoConfig
    Redis     *RedisConfig
    NATS      *NATSConfig
    Whitelist *WhitelistConfig
}

type NATSConfig struct {
    URL    string
    Queues []NATSQueueConfig
}

type NATSQueueConfig struct {
    Name          string  // 队列名称
    StreamName    string  // JetStream Stream 名
    StreamSubject string  // 订阅主题
    ConsumerName  string  // Consumer 名
    Handler       string  // 处理器注册名
    WorkerCount   int     // 工作协程数
    BatchSize     int     // 批量拉取数
    AckWait       int     // 确认等待秒数
    MaxDeliver    int     // 最大重发次数
}
```

### 4.5 错误处理 (`common/codeError/`)

所有函数返回 `*codeError.CodeError`，禁止使用标准 `error`。

```go
type CodeError struct { code int32; msg string }

func New(code int32, msg string) *CodeError
func (e *CodeError) Msg(msg string) *CodeError  // 不可变派生
func (e *CodeError) Error() string               // 实现 error 接口
func (e *CodeError) GetCode() int32
func (e *CodeError) GetMsg() string
```

预定义错误码：
| 错误码 | 常量 | 含义 |
|--------|------|------|
| 0 | Success | 成功 |
| 200 | HttpSuccess | HTTP 成功 |
| 1001 | SystemError | 系统错误 |
| 1002 | ConfigError | 配置错误 |
| 1003 | DBError | 数据库错误 |
| 1004 | RedisError | Redis 错误 |
| 1005 | LoggerError | 日志错误 |
| 1006 | NATSInitError | NATS 初始化错误 |
| 1007 | NATSError | NATS 错误 |
| 1008 | StreamError | Stream 错误 |
| 1009 | ConsumerError | Consumer 错误 |
| 1010 | UnknownCmd | 未知命令 |

使用模式：
```go
return codeError.RedisError.Msg("dao.Init redis.Ping error: " + err.Error())
```

## 5. 数据流

### 5.1 HTTP 请求流

```
Gate --[POST /api/gateway]--> HandleGateway
  │ Bind(gateway.Request)
  v
service.RouteCmd(cmd, uid, data)
  │ switch cmd
  v
PingService(uid)
  │
  v
gateway.Response{Code:0, Message:"success", Data:{...}}
```

### 5.2 NATS 消息流

```
外部发布者 --[NATS Publish]--> Stream "GATEWAY" / Subject "gateway.request.*"
  │
  v
GatewayHandler.addSubscription()
  │ 1. GetHandler("RouteCmd")
  │ 2. EnsureStream (Stream + Consumer)
  │ 3. PullSubscribe
  │ 4. NewWorkerPool(8 workers)
  │ 5. go dispatcher()
  v
dispatcher()
  │ sub.Fetch(batchSize=16) 循环拉取
  │ 每条消息 -> pool.Submit(msg)
  v
WorkerPool.worker()
  │ 从 jobChan 取消息
  v
processMsg()
  │ 1. json.Unmarshal -> gateway.Request
  │ 2. handler(cmd, uid, data)  // 即 service.RouteCmd
  │ 3. msg.Respond(response)
  │ 4. msg.Ack()
  │ panic: msg.Nak()
```

## 6. 多队列 NATS 架构

`GatewayHandler` 支持同时订阅多个队列，每个队列拥有独立的：
- JetStream Stream + Pull Consumer
- WorkerPool（可配置不同的 worker 数量）
- dispatcher 协程
- 处理函数（通过 Handler 注册表绑定）

配置示例：
```yaml
nats:
  url: "nats://127.0.0.1:4222"
  queues:
    - name: "gatewaymodel"
      streamName: "GATEWAY"
      streamSubject: "gatewaymodel.request"
      consumerName: "lobby-gatewaymodel"
      handler: "RouteCmd"
      workerCount: 8
      batchSize: 16
      ackWait: 30
      maxDeliver: 3
    - name: "message"
      streamName: "MESSAGE"
      streamSubject: "message.push"
      consumerName: "lobby-message"
      handler: "MessageCmd"
      workerCount: 4
      batchSize: 8
      ackWait: 15
      maxDeliver: 5
```

初始化流程：
```
main.go
  ├─ service.RegisterHandler("RouteCmd", service.RouteCmd)
  ├─ service.RegisterHandler("MessageCmd", service.MessageCmd)  // 可选
  └─ NewGatewayHandler(nc, cfg.NATS.Queues, log)
       └─ for each queue:
            ├─ GetHandler(cfg.Handler)
            ├─ EnsureStream(streamCfg)
            ├─ PullSubscribe(subject, consumer)
            ├─ NewWorkerPool(count, handler)
            ├─ pool.Start()
            └─ go dispatcher(name, sub, batchSize, pool)
```

## 7. 依赖关系

```
main ─────────────────────────────────────────────────
  ├── config                    (无内部依赖)
  ├── common/logger
  │   ├── config
  │   └── common/codeError
  ├── db
  │   ├── config
  │   └── common/codeError
  ├── dao
  │   ├── config
  │   └── common/codeError
  ├── nats
  │   ├── config
  │   ├── common/codeError
  │   └── model/nats
  ├── handler/http
  │   ├── model/gateway
  │   └── service
  ├── handler/nats
  │   ├── config
  │   ├── common/codeError
  │   ├── nats (本地包)
  │   ├── model/nats, model/gateway
  │   └── service
  └── service
      └── common/codeError
```

外部依赖：
- `github.com/labstack/echo/v4` — HTTP 框架
- `github.com/nats-io/nats.go` — NATS JetStream 客户端
- `github.com/redis/go-redis/v9` — Redis 客户端
- `github.com/spf13/viper` — 配置加载
- `go.mongodb.org/mongo-driver/v2` — MongoDB 驱动
- `go.uber.org/zap` — 结构化日志

## 8. 扩展指南

### 新增 NATS 队列

1. `config.yaml` 的 `queues` 列表新增一项，指定 `handler` 名称
2. `service/` 下实现新的 `HandlerFunc` 业务函数
3. `main.go` 中 `service.RegisterHandler("名称", 函数)` 注册

### 新增 HTTP 回调

1. `handler/http/` 实现处理器
2. `main.go` 中通过 `e.POST/GET(...)` 注册路由

### 新增 C2S 命令

1. `service/` 下实现业务函数
2. `service/router.go` 的 `RouteCmd` 中 `switch cmd` 添加分支

### 新增 Redis 缓存模型

1. 定义结构体，实现 `dao.DBData` 接口（`RedisKey`, `Pack`, `UnPack`）
2. 通过 `dao.FillDBInfo` / `dao.SetDBInfo` / `dao.DelDBInfo` 读写
