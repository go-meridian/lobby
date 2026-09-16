# AGENTS.md

This file provides guidance to the AI agent when working with code in this repository.

## 项目概述

Lobby 是游戏大厅业务服务，与 Gate（WebSocket 网关）组成完整服务。

```
客户端 --[WS]--> Gate:8080 --[HTTP POST JSON]--> Lobby:9001
                                  /api/gateway
```

- **Gate** (`D:/self/go/web/Gate/`): 纯转发层，维护 WS 连接和会话，无业务逻辑，无数据库
- **Lobby**: 业务逻辑层，处理 cmd 路由、数据读写

技术栈：Echo v4 + GORM (MySQL) + go-redis + zap 日志。

## 架构分层

```
Gate --HTTP POST--> handler/http/ --> service/ --> dao/
                                       ↑
handler/c2s/ (MQ, 预留) ----------------┘
```

- **Gateway 模式**: Gate 转发 `GatewayRequest{SessionID, UID, Cmd, Data}` 到 `POST /api/gateway`，Lobby 侧 `service.RouteCmd(cmd, uid, data)` 按 cmd 路由
- **共享结构体**: `GatewayRequest` / `GatewayResponse` 在 Gate (`client/lobby.go`) 和 Lobby (`handler/http/gateway.go`) 各自定义，字段和 json tag 必须一致
- **UID 分表**: `dao.TableByUid(uid, table)` 按 UID 分片（每 200 万分一个表）
- **Redis 缓存**: 数据模型实现 `dao.DBData` 接口（`RedisKey/Pack/UnPack`），通过 `FillDBInfo/SetDBInfo` 读写

## 错误处理

返回类型使用 `*base.CodeError`（在 `common/base/` 中定义），禁止使用标准 `error` 接口。

错误赋值规范：
```go
// 正确：先赋值 err，再打日志
err = errorcode.SystemError.Msg(e.Error())
llog.Error("xxx error! err[ %s ]", err.Error())
return err

// 错误：直接返回或用原始错误打日志
return errorcode.SystemError.Msg(e.Error())  // 禁止
llog.Error("xxx error! err[ %s ]", e.Error())  // 禁止
```

## 日志

使用 `zap` 结构化日志，禁止 `fmt.Println` 调试。

## 常量

放在 `model/constant/` 下，按领域建文件。

## Protobuf 工作流

修改 proto 必须先改源文件，运行生成脚本后复制 `.pb.go` 到 `common/proto/`。禁止直接修改生成文件。

## 配置

`config.yaml` 通过 viper 加载，结构体定义在 `config/config.go`。

## 构建验证

```bash
go build ./...
```

## 禁止事项

- 禁止 emoji 出现在代码、注释、文档中
- 禁止 `_ = someFunc()`，所有错误必须显式处理
- 禁止硬编码密钥/Token/密码
