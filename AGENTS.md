# AGENTS.md

本文件定义 Lobby 项目的架构规范和开发约束，AI Agent 必须遵守。

## 项目概述

Lobby 是游戏大厅业务服务，与 Gate（WebSocket 网关）组成完整服务。

```
客户端 --[WS]--> Gate:8080 --[NATS Core]--> Lobby:9001
                                  /api/gateway (HTTP 降级)
```

- **Gate** (`D:/self/go/web/Gate/`): WS 网关，维护连接和会话，无业务逻辑
- **Lobby**: 业务逻辑层，处理 MsgId 路由、数据读写

技术栈：Echo v4 + MongoDB (mongo-driver v2) + Redis (go-redis v9) + NATS Core + Protobuf + zap 日志。

## 架构分层

```
main.go
  ├── config/               配置加载 (viper)
  ├── db/                   MongoDB 连接
  ├── dao/                  Redis 缓存
  │   ├── dao.go            Init()、Redis 客户端
  │   └── cache.go          DBData 接口、FillDBInfo/SetDBInfo/DelDBInfo
  ├── mq/                   MQ 基础设施
  │   ├── init.go           client、log、Init()、IsConnected()
  │   ├── publisher.go      Publisher、Publish()、GetPublisher()
  │   ├── subscription.go   订阅注册表、RegisterCoreSubscription()、Start()
  │   └── protocol.go       Handler 类型、handleMessage()
  ├── model/                结构体定义 + 方法
  │   ├── codeerror/        CodeError 错误码
  │   ├── proto/            Protobuf 生成文件
  │   │   ├── common/       ErrMsg, OnlineNum
  │   │   ├── account/      LoginReq/Rsp, ServiceToken
  │   │   ├── health/       HealthCheckReq/Rsp, ServiceInfo
  │   │   ├── gate/         GateRequest/Response/Push, AuthReq/Rsp
  │   │   └── packet/       MsgId 枚举
  │   ├── httpmodel/        HTTP API 请求/响应结构体
  │   └── eventmodel/       事件模型定义 (HealthEvent 等)
  ├── handler/              入口层
  │   ├── mqhandler/        MsgId 路由 + handler 注册
  │   │   ├── init.go       MsgHandler、Register()、RouteMsg()、Init()
  │   │   └── health.go     heartbeatHandler
  │   ├── httphandler/      HTTP 入口
  │   │   ├── init.go       Init()、Register()
  │   │   ├── health.go     HandleHealthFunc()
  │   │   ├── checks.go     checkMongoDB/checkRedis/checkNATS
  │   │   └── middleware.go  RequestID/AccessLog/Recover/RateLimit
  │   ├── electhandler/     选主机制 (etcd/redis 后端)
  │   └── eventhandler/     事件处理 (health 等)
  └── service/              业务逻辑
      └── init.go           Publisher 接口、Init()、GetPublisher()
```

### 分层职责

| 层 | 职责 | 可依赖 |
|---|---|---|
| model/ | 结构体定义、方法、接口 | codeerror |
| mq/ | MQ 客户端、Publisher、订阅管理、协议编解码 | model/proto |
| handler/ | MsgId 路由、HTTP 入口、中间件 | model, mq |
| service/ | 业务逻辑函数 | model |
| db/dao | 基础设施封装 | config, model/codeerror |

### 依赖方向

```
config ← logger ← db/dao ← model ← mq ← handler/mqhandler ← main
                             ↑                ↑
                         service ←────────────┘
```

## Protobuf 工作流

Proto 源文件在独立项目 `D:\self\go\web\proto\`，导出文件在 `model/proto/`。

修改流程：`proto/` 项目修改 .proto → 运行 `gen.bat` → 复制到 `model/proto/`

pb.go 中 import 路径需手动修复：`proto/common` → `github.com/go-meridian/lobby/model/proto/common`

### Gate 协议

Gate 与 Lobby 之间使用 Protobuf 二进制通信：

| 消息 | 方向 | 说明 |
|------|------|------|
| GateRequest | Gate→Lobby | ConnId, RequestId, MsgId, Payload |
| GateResponse | Lobby→Gate | ConnId, RequestId, MsgId, Payload |
| GatePush | Lobby→Gate | ConnId, MsgId, Payload（主动推送） |

### MsgId 枚举

| MsgId | 说明 |
|-------|------|
| 1001 | MSG_HEARTBEAT_REQ 心跳请求 |
| 1002 | MSG_HEARTBEAT_RSP 心跳响应 |
| 2001 | MSG_LOGIN_REQ 登录请求 |
| 2002 | MSG_LOGIN_RSP 登录响应 |
| 2003 | MSG_LOGIN_TEST_REQ 测试登录 |
| 2004 | MSG_LOGIN_TEST_RSP 测试登录响应 |

## NATS 通信

| 方向 | 模式 | Subject | 说明 |
|------|------|---------|------|
| Gate→Lobby | NATS Core Request/Reply | `gate2lobby.*` | 同步，Protobuf |
| Lobby→Gate | NATS Core Publish | `lobby2gate.{msgId}` | 异步，Protobuf GatePush |

### 路由流程

```
NATS 消息 → mq.handleMessage (proto.Unmarshal GateRequest)
  → mqhandler.RouteMsg (根据 msgId 查找 msgRegistry)
    → MsgHandler (service 业务处理)
      → mq.handleMessage 构造 GateResponse (proto.Marshal) → ReplyTo
```

## 注册模式

### MsgId 注册

```go
// handler/mqhandler/init.go
func Init(l *logger.Logger) {
    log = l
    mq.RegisterCoreSubscription("gate2lobby.*", RouteMsg, 8)
    mq.RegisterPublishStream("LOBBY2GATE", "lobby2gate")
    register()
}

func register() {
    Register(uint32(packet.MsgId_MSG_HEARTBEAT_REQ), heartbeatHandler)
}
```

### HTTP 路由注册

```go
// handler/httphandler/init.go
func Register(e *echo.Echo) {
    e.GET("/health", HandleHealthFunc())
}
```

## 日志模式

所有包统一使用包级 `var log *logger.Logger`，通过 `Init(l *logger.Logger)` 初始化。

- 禁止将 log 作为函数参数传递
- 禁止在结构体中持有 log 字段

```go
package xxx

var log *logger.Logger

func Init(l *logger.Logger) {
    log = l
}
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
mqClient, _ := mqLib.NewClient(mqCfg)

// 4. MQ + Handler 层（顺序重要：mq.Init 先于 mqhandler.Init）
mq.Init(mqClient, log)        // MQ 基础设施
mqhandler.Init(log)            // 注册订阅 + MsgId handler
httpHandler.Init(log)

// 5. Service 层
service.Init(log, mq.GetPublisher())

// 6. 启动订阅
mq.Start()

// 7. 启动 HTTP
httpHandler.Register(e)
```

## 错误处理

返回类型使用 `*codeerror.CodeError`，禁止使用标准 `error` 接口。

```go
// 正确：先赋值 err，再打日志
err = codeerror.SystemError.Msg(e.Error())
llog.Error("xxx error! err[ %s ]", err.Error())
return err
```

## Job 定时任务

使用外部库 `github.com/go-meridian/job` 管理定时任务。选主成功后通过 `WithOnLeader` 回调启动，`WithOnDemote` 停止。

## Elect 选主机制

路径：`handler/electhandler`，支持 etcd 和 redis 两种后端。通过 blank import 注册后端。

## 中间件

按注册顺序执行：Recover → RequestID → RateLimit → AccessLog

### 请求 ID 链路追踪

```
HTTP: middleware 生成 → echo.Context → RouteMsg → MsgHandler
NATS: Gate 携带 req.RequestId → handleMessage 提取（空则自动生成）
```

## 常量

放在 `model/constant/` 下，按领域建文件。

## 配置

`config.yaml` 通过 viper 加载，结构体定义在 `config/config.go`。

## 构建验证

```bash
go build ./...
go vet ./...
```

## Docker 部署

| 文件 | 用途 |
|------|------|
| `Dockerfile` | 多阶段构建镜像 |
| `config.docker.yaml` | Docker 环境配置 |
| `docker-compose.yml` | 完整服务（Lobby + 中间件） |
| `docker-compose.infra.yml` | 所有中间件 |
| `docker-compose.local.yml` | 仅 Lobby（连接宿主机中间件） |
| `make.bat` / `Makefile` | 快捷命令 |

常用命令：`make.bat all-up` / `make.bat local-up` / `make.bat infra-up` / `make.bat down`

## 禁止事项

- 禁止 emoji 出现在代码、注释、文档中
- 禁止 `_ = someFunc()`，所有错误必须显式处理
- 禁止硬编码密钥/Token/密码
- 禁止 `fmt.Println` 调试
- 禁止直接调用 `time.Now()` 等时间函数（必须通过 `common/util/utilTime.go` 封装）

## 代码组织

- 小文件优先：宁可多个小文件，不要少数大文件
- 高内聚低耦合：每个文件聚焦单一领域
- 文件大小限制：典型 200-400 行，单文件不超过 800 行
- 按领域组织：按功能/业务域划分目录结构

## 新增命令流程

### 新增 MsgId 命令

1. proto 项目定义 .proto 消息 + MsgId 枚举
2. 复制 .pb.go 到 `model/proto/`，修复 import 路径
3. 在 `handler/mqhandler/init.go` 的 `register()` 中调用 `Register(msgId, handler)`
4. handler 签名：`func(connId uint64, requestId uint64, payload []byte) ([]byte, *codeerror.CodeError)`

### 新增 HTTP 路由

1. `handler/httphandler/` 下实现处理函数
2. 在 `init.go` 的 `Register()` 中注册路由
