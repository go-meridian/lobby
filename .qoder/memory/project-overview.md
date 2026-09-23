# 项目概述

## 服务定位

Lobby 是游戏大厅业务服务，与 Gate（WebSocket 网关）组成完整服务。

```
客户端 --[WS]--> Gate:8080 --[NATS Core]--> Lobby:9001
                                  /api/gateway (HTTP 降级)
```

- **Gate** (`D:/self/go/web/Gate/`): WS 网关，维护连接和会话，无业务逻辑
- **Lobby**: 业务逻辑层，处理 MsgId 路由、数据读写

## 技术栈

| 组件 | 技术 |
|------|------|
| Web 框架 | Echo v4 |
| 数据库 | MongoDB (mongo-driver v2) |
| 缓存 | Redis (go-redis v9) |
| 消息队列 | NATS Core |
| 序列化 | Protobuf (google.golang.org/protobuf) |
| 日志 | go-meridian/logger v1.0.4（zap 封装 + 文件轮转 + key=value 编码器） |
| 配置 | viper |

## 目录结构

```
main.go
  ├── config/               配置加载 (viper)
  ├── db/                   MongoDB 连接
  ├── dao/                  Redis 缓存
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
  │   ├── mqhandler/        NATS 入口 (MsgId 路由 + 订阅管理)
  │   │   ├── init.go       MQ 客户端、Publisher、handleMessage
  │   │   ├── router.go     MsgHandler、Register、RouteMsg、msgRegistry
  │   │   └── register.go   订阅注册 + MsgId 处理函数注册
  │   ├── httphandler/      HTTP 入口
  │   ├── electhandler/     选主机制 (etcd/redis 后端)
  │   └── eventhandler/     事件处理 (health 等)
  └── service/              业务逻辑
```

## 分层职责

| 层 | 职责 | 可依赖 |
|---|---|---|
| model/ | 结构体定义、方法、接口 | codeerror |
| handler/ | 入口注册、MsgId 路由、中间件 | model, service |
| service/ | 业务逻辑函数 | model |
| db/dao | 基础设施封装 | config, model/codeerror |

## 依赖方向

```
config ← logger ← db/dao ← model ← handler/mqhandler ← main
                              ↑
                          service ← handler/mqhandler
```

## Proto import 路径修复

proto 生成的 .pb.go 文件 import 路径为 `proto/common` 等相对路径，在 lobby 项目中需改为完整路径：

```go
// 修复前
import common "proto/common"
// 修复后
import common "github.com/go-meridian/lobby/model/proto/common"
```
