# 项目概述

## 服务定位

Lobby 是游戏大厅业务服务，与 Gate（WebSocket 网关）组成完整服务。

```
客户端 --[WS]--> Gate:8080 --[NATS Core]--> Lobby:9001
                                  /api/gateway (HTTP 降级)
```

- **Gate** (`D:/self/go/web/Gate/`): WS 网关，维护连接和会话，无业务逻辑
- **Lobby**: 业务逻辑层，处理 cmd 路由、数据读写

## 技术栈

| 组件 | 技术 |
|------|------|
| Web 框架 | Echo v4 |
| 数据库 | MongoDB (mongo-driver v2) |
| 缓存 | Redis (go-redis v9) |
| 消息队列 | NATS Core |
| 日志 | zap + 日志轮转 |
| 配置 | viper |

## 目录结构

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

## 分层职责

| 层 | 职责 | 可依赖 |
|---|---|---|
| model/ | 结构体定义、方法、接口 | codeerror |
| handler/ | 入口注册、cmd 路由、中间件 | model, service |
| service/ | 业务逻辑函数 | model |
| db/dao/nats | 基础设施封装 | config, model/codeerror |

## 依赖方向

```
config ← logger ← db/dao/nats ← model ← handler ← main
                                  ↑
                              service ← handler
```

- model/ 可依赖 service/（model/http/handler.go 调用 service 函数）
- handler/ 依赖 model/（使用结构体）和 service/（注册业务函数）
- service/ 不依赖 handler/
