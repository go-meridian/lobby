# AGENTS.md

This file provides guidance to the AI agent when working with code in this repository.

## 项目概述

Go 错误码库 (`github.com/go-meridian/codeerror`)，提供带错误码的 `CodeError` 类型，Go 1.16+。被 Lobby 项目作为基础错误类型依赖。

## 核心设计

- `*CodeError` 实现了 `error` 接口，所有方法对 nil receiver 安全（返回零值）
- `Msg()` 返回新对象（不可变语义），`SetMsg()` 原地修改（可变语义）
- `New(code, msg)` 构造函数，`Error()` 返回 msg 文本

## Lobby 项目集成方式

### re-export 层 (`model/codeerror/codeerror.go`)

```go
package codeerror
import ce "github.com/go-meridian/codeerror"
type CodeError = ce.CodeError  // 类型别名，透明暴露
var New = ce.New
```

项目统一 `import lobby/model/codeerror`，不直接导入外部包。

### 错误码定义 (`model/codeerror/template.go`)

```go
var (
    Success       = ce.New(0, "success")
    HttpSuccess   = ce.New(200, "success")
    SystemError   = ce.New(1001, "system error")
    ConfigError   = ce.New(1002, "config error")
    DBError       = ce.New(1003, "database error")
    RedisError    = ce.New(1004, "redis error")
    LoggerError   = ce.New(1005, "logger error")
    NATSInitError = ce.New(1006, "NATS init error")
    NATSError     = ce.New(1007, "NATS error")
    StreamError   = ce.New(1008, "stream error")
    ConsumerError = ce.New(1009, "consumer error")
    UnknownCmd    = ce.New(1010, "unknown cmd")
)
// 业务错误码范围：2000-9999
```

### 三种使用模式

**1. 创建错误 — `codeerror.XXX.Msg("detail")`**

```go
return codeerror.DBError.Msg("db.Init mongo.Connect error: " + err.Error())
return codeerror.RedisError.Msg("dao.Init redis.Ping error: " + err.Error())
```

所有初始化/业务函数返回 `*codeerror.CodeError`，禁止返回标准 `error`。

**2. 消费错误码 — `GetCode()` / `GetMsg()`**

```go
// HTTP 响应
return c.JSON(http.StatusOK, &Response{Code: int(ce.GetCode()), Message: ce.GetMsg()})
// NATS 响应
s.replyError(msg, requestID, int(ce.GetCode()), ce.GetMsg())
```

**3. 日志记录 — `Error()`**

```go
zap.String("error", ce.Error())
```

### 函数签名规范

```go
func Init(cfg *config.Config) *codeerror.CodeError
func RouteCmd(...) (interface{}, *codeerror.CodeError)
```

## 编码约束

- 包级别零依赖，禁止引入第三方库
- Go 1.16 兼容（无泛型、无 errors.Is/As 依赖）
- 修改此库后需更新 Lobby 的 vendor 目录
