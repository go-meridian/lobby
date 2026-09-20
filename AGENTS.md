# AGENTS.md

本文件定义 Lobby 项目的架构规范和开发约束，AI Agent 必须遵守。

## 项目概述

Lobby 是游戏大厅业务服务，与 Gate（WebSocket 网关）组成完整服务。

```
客户端 --[WS]--> Gate:8080 --[NATS Core]--> Lobby:9001
                                  /api/gateway (HTTP 降级)
```

- **Gate** (`D:/self/go/web/Gate/`): WS 网关，维护连接和会话，无业务逻辑
- **Lobby**: 业务逻辑层，处理 cmd 路由、数据读写

技术栈：Echo v4 + MongoDB (mongo-driver v2) + Redis (go-redis v9) + NATS Core + zap 日志。

## 架构分层

```
main.go
  ├── config/               配置加载 (viper)
  ├── common/logger/        日志系统 (zap + 日志轮转)
  ├── db/                   MongoDB 连接
  ├── dao/                  Redis 缓存
  ├── nats/                 NATS 客户端
  ├── model/                结构体定义 + 方法
  │   ├── codeerror/        CodeError 错误码
  │   ├── gateway/          Request/Response 消息体
  │   ├── http/             APIHandler
  │   └── nats/             GatewayHandler + WorkerPool + NATSPublisher + CoreSubscriber
  ├── handler/              入口层（路由注册 + cmd 映射）
  │   ├── router.go         RouteCmd + cmd 注册表
  │   ├── httphandler/      HTTP 入口
  │   └── nats/             NATS 入口
  └── service/              业务逻辑
```

### 分层职责

| 层 | 职责 | 可依赖 |
|---|---|---|
| model/ | 结构体定义、方法、接口 | codeerror |
| handler/ | 入口注册、cmd 路由、中间件 | model, service |
| service/ | 业务逻辑函数 | model |
| db/dao/nats | 基础设施封装 | config, model/codeerror |

### 依赖方向

```
config ← logger ← db/dao/nats ← model ← handler ← main
                                  ↑
                              service ← handler
```

- model/ 可依赖 service/（model/http/handler.go 调用 service 函数）
- handler/ 依赖 model/（使用结构体）和 service/（注册业务函数）
- service/ 不依赖 handler/

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
    Publish(subject string, data []byte) *codeerror.CodeError      // 异步，不等待 flush
    PublishSync(subject string, data []byte) *codeerror.CodeError  // 同步，等待 flush 确认
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

## 错误处理

返回类型使用 `*codeerror.CodeError`（在 `model/codeerror/` 中定义），禁止使用标准 `error` 接口。

```go
// 正确：先赋值 err，再打日志
err = codeerror.SystemError.Msg(e.Error())
llog.Error("xxx error! err[ %s ]", err.Error())
return err

// 禁止：直接返回或用原始错误打日志
return codeerror.SystemError.Msg(e.Error())
llog.Error("xxx error! err[ %s ]", e.Error())
```

## 日志

使用 `zap` 结构化日志，禁止 `fmt.Println` 调试。日志中必须包含 `requestID` 字段。

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

项目支持 Docker 容器化部署，相关文件：

| 文件 | 用途 |
|------|------|
| `Dockerfile` | 多阶段构建镜像 |
| `config.docker.yaml` | Docker 环境配置 |
| `docker-compose.yml` | 完整服务（Lobby + 中间件） |
| `docker-compose.mongo.yml` | 仅 MongoDB |
| `docker-compose.redis.yml` | 仅 Redis |
| `docker-compose.nats.yml` | 仅 NATS |
| `docker-compose.infra.yml` | 所有中间件 |
| `docker-compose.local.yml` | 仅 Lobby（连接宿主机中间件） |
| `Dockerfile.local` | 本地模式构建 |
| `config.local.yaml` | 本地模式配置 |
| `check-infra.bat` | 检查并启动中间件脚本 |
| `Makefile` | Linux/Mac 快捷命令 |
| `make.bat` | Windows 快捷命令 |

### 部署模式

**模式1：全容器化（推荐新环境）**
```cmd
.\make.bat all-up
```
启动 Lobby + MongoDB + Redis + NATS，所有服务独立容器。

**模式2：本地模式（连接已有中间件）**
```cmd
# 检查中间件状态，未运行则自动启动
.\make.bat check-infra

# 启动 Lobby（连接宿主机中间件）
.\make.bat local-up
```
仅启动 Lobby 容器，连接宿主机已运行的 MongoDB/Redis/NATS。

### 常用命令

**Windows (PowerShell/cmd):**
```cmd
# 全容器化启动
.\make.bat all-up

# 本地模式（连接宿主机中间件）
.\make.bat local-up
.\make.bat local-down

# 检查并启动中间件（如果没有运行）
.\make.bat check-infra

# 仅启动中间件
.\make.bat infra-up

# 启动单个中间件
.\make.bat mongo-up
.\make.bat redis-up
.\make.bat nats-up

# 停止服务
.\make.bat down
.\make.bat infra-down
```

**Linux/Mac (bash):**
```bash
make all-up
make infra-up
make down
```

### 网络配置

Docker 容器使用 `lobby-network` 桥接网络，服务间通过容器名访问：
- MongoDB: `mongo:27017`
- Redis: `redis:6379`
- NATS: `nats:4222`

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

### 新增 Service 命令

1. `service/` 下新建文件，实现处理函数
2. 在 `init()` 中调用 `handler.Register("CMD", YourService)`

### 新增 HTTP 路由

1. `handler/httphandler/` 下实现处理函数
2. 在 `register.go` 的 `Register()` 中注册路由

### 新增 NATS 队列

1. `handler/nats/register.go` 的 `init()` 中调用 `RegisterCoreSubscription` 或 `RegisterPublishStream`
