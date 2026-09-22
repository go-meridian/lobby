# 架构约定

## Protobuf 工作流

### 文件位置

| 类型 | 路径 | 说明 |
|------|------|------|
| Proto 源文件 | `D:\self\go\web\proto\` | 独立项目，定义所有 .proto 文件 |
| 导出文件 | `model/proto/` | .pb.go 生成文件 |

### Proto 包结构

```
model/proto/
├── common/      common.ErrMsg, common.OnlineNum
├── account/     account.LoginReq/LoginRsp/LoginTestReq/LoginTestRsp/ServiceToken
├── health/      health.HealthCheckReq/HealthCheckRsp/ServiceInfo/ServiceStatus
├── gate/        gate.GateRequest/GateResponse/GatePush/AuthReq/AuthRsp/ConnOpenNotify/ConnCloseNotify
└── packet/      packet.MsgId 枚举
```

### Gate 协议消息

Gate 与 Lobby 之间使用 Protobuf 二进制通信：

```go
// Gate → Lobby 请求
type GateRequest struct {
    ConnId    uint64  // WebSocket 连接 ID
    RequestId uint64  // 请求 ID（匹配响应）
    MsgId     uint32  // 消息类型 ID（packet.MsgId 枚举）
    Payload   []byte  // 业务 Proto 序列化字节
}

// Lobby → Gate 响应
type GateResponse struct {
    ConnId    uint64
    RequestId uint64
    MsgId     uint32
    Payload   []byte
}

// Lobby → Gate 主动推送（无 requestId）
type GatePush struct {
    ConnId  uint64  // 0=广播
    MsgId   uint32
    Payload []byte
}
```

### MsgId 枚举

```go
// packet/MsgId.pb.go
MSG_HEARTBEAT_REQ  = 1001  // 心跳请求
MSG_HEARTBEAT_RSP  = 1002  // 心跳响应
MSG_LOGIN_REQ      = 2001  // 登录请求
MSG_LOGIN_RSP      = 2002  // 登录响应
MSG_LOGIN_TEST_REQ = 2003  // 测试登录请求
MSG_LOGIN_TEST_RSP = 2004  // 测试登录响应
```

### 工作流程

1. 在 `D:\self\go\web\proto\` 修改 .proto 文件
2. 运行 `gen.bat` 生成 .pb.go
3. 复制到 `model/proto/` 目录
4. 若 pb.go 中 import 路径为 `proto/common` 等相对路径，需手动改为 `github.com/go-meridian/lobby/model/proto/common`

## NATS 通信

| 方向 | 模式 | Subject | 说明 |
|------|------|---------|------|
| Gate→Lobby | NATS Core Request/Reply | `gate2lobby.*` | 同步，Protobuf GateRequest/GateResponse |
| Lobby→Gate | NATS Core Publish | `lobby2gate.{msgId}` | 异步，Protobuf GatePush |
| Gate→Lobby | HTTP POST | `/api/gateway` | 降级方案 |

### NATSClient 接口

```go
type NATSClient interface {
    Publish(subject string, data []byte) *codeerror.CodeError      // 异步
    PublishSync(subject string, data []byte) *codeerror.CodeError  // 同步，等待 flush
    Subscribe(subject string, handler natsLib.MsgHandler) (*natsLib.Subscription, error)
}
```

## MsgId 路由体系

### 函数签名

```go
// mqhandler/ 层：NATS 订阅处理（含 msgId 用于路由分发）
type MQHandler func(connId uint64, requestId uint64, msgId uint32, payload []byte) ([]byte, *codeerror.CodeError)

// mqhandler/ 层：Service 处理函数（msgId 已由 RouteMsg 消费）
type MsgHandler func(connId uint64, requestId uint64, payload []byte) ([]byte, *codeerror.CodeError)
```

### 路由流程

```
NATS 消息 → handleMessage (proto.Unmarshal GateRequest)
  → RouteMsg (根据 msgId 查找 msgRegistry)
    → MsgHandler (service 业务处理)
      → 返回 payload ([]byte)
        → handleMessage 构造 GateResponse (proto.Marshal) → ReplyTo
```

### MsgId 注册

```go
// handler/mqhandler/init.go
func init() {
    RegisterCoreSubscription("gate2lobby.*", RouteMsg, 8)
    RegisterPublishStream("LOBBY2GATE", "lobby2gate")
    Register(uint32(packet.MsgId_MSG_HEARTBEAT_REQ), heartbeatService)
}
```

### 新增 MsgId 命令

1. 在 proto 项目定义 .proto 消息 + MsgId 枚举
2. 复制 .pb.go 到 `model/proto/`
3. 在 `../../handler/mqhandler/init.go` 的 `init()` 中调用 `Register(msgId, handler)`
4. handler 函数签名：`func(connId uint64, requestId uint64, payload []byte) ([]byte, *codeerror.CodeError)`

## HTTP Handler 模式

### Handler 实现

```
handler/httphandler/
├── init.go       # Init(l) + Register(e) 路由注册
├── health.go     # HandleHealthFunc()
└── middleware.go  # RequestID / AccessLog / Recover / RateLimit
```

### 新增 HTTP 路由

1. `handler/httphandler/` 下实现处理函数
2. 在 `init.go` 的 `Register()` 中注册路由

## 中间件

按注册顺序执行：

1. **Recover** - panic 恢复，日志包含 requestID 和 stack
2. **RequestID** - 生成唯一请求 ID，格式 `http_{seq}_{ts}`
3. **RateLimit** - 基于 IP 限流，100 req/s，突发 200
4. **AccessLog** - 请求日志，包含 requestID/method/uri/status/latency

### 请求 ID 链路追踪

```
HTTP: middleware 生成 → echo.Context → RouteMsg → MsgHandler
NATS: Gate 携带 req.RequestId → handleMessage 提取（空则自动生成 uint64 递增 ID）
```

## 初始化顺序

```go
// 1. 基础设施
config.Init()
logger.Init(cfg)

// 2. 存储层
db.Init(cfg)
dao.Init(cfg)

// 3. MQ 客户端
mqClient, _ := mq.NewClient(mqCfg)

// 4. Handler 层
httpHandler.Init(log)
natsHandler.Init(mqClient, log)
electhandler.Init(cfg, log)

// 5. Service 层
service.Init(log, natsHandler.GetPublisher())

// 6. 启动订阅
natsHandler.Start()

// 7. 启动 HTTP
httpHandler.Register(e)
```

## 日志模式

所有包统一使用包级 `var log *logger.Logger`，通过 `Init(l *logger.Logger)` 初始化。禁止将 log 作为函数参数传递。

```go
// 每个包的标准模式
package xxx

var log *logger.Logger

func Init(l *logger.Logger) {
    log = l
}
```

## Job 定时任务

使用外部库 `github.com/go-meridian/job` 管理定时任务。选主成功后通过 `WithOnLeader` 回调启动，`WithOnDemote` 停止。

## Elect 选主机制

路径：`handler/electhandler`，支持 etcd 和 redis 两种后端。通过 blank import 注册后端。

## Event 事件系统

| 层 | 路径 | 职责 |
|---|---|---|
| model/eventmodel/ | 事件结构体定义 | `HealthEvent`、`TopicHealth` 等 |
| handler/eventhandler/ | 事件处理函数 | `onHealth(evt)` 等 |
