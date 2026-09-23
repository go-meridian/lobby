# AGENTS.md

This file provides guidance to the AI agent when working with code in this repository.

## Project Overview

Go library package (not a binary). Wraps `go.uber.org/zap` with a custom `LogWriter` that rotates log files by both date and size. 同时导出 zap 的字段构造函数和 Logger 封装，消费方无需额外引入 zap 包。

## Conventions

- Package name: `logger` — all files use `package logger`
- Comments and documentation in Chinese
- No `main` package — this is imported by other projects
- No tests exist yet — when adding tests, use `_test.go` files in the same package

## Key Design

- `LogWriter` (writer.go) 是核心：互斥锁保护，日期 + 大小双维度轮转，自动清理超过 `maxAge` 天的旧文件
- 文件命名：`{prefix}.{YYYYMMDD}.{seq}`（如 `app.20260920.1`）
- 全局单例：`Init()` 设置包级 `*zap.Logger`，通过 `Get()` 无锁原子读取（`atomic.Value`，永不返回 nil，未初始化时为 `zap.NewNop()`）
- `Close()` 幂等（`sync.Once`）：刷新缓冲区、关闭主/错误写入器后重置为 Nop
- 错误日志独立输出：`Config.ErrorFile` 非空时创建独立 `LogWriter` + error 级别 enabler，通过 `zapcore.NewTee(mainCore, errorCore)` 组合双 core
- `field.go` 导出 zap 字段构造函数（`String`, `Int`, `Error`, `Any` 等），消费方用 `logger.String(...)` 代替 `zap.String(...)`
- `zap.go` 提供 `Logger` 封装类型，通过 `L()` 获取全局实例或 `Wrap(zapLogger)` 包装已有实例
- `encoder.go` 自定义 `keyValueEncoder`（实现完整 `zapcore.Encoder`），输出 `DATE[时间] 大写级别 消息 key=value` 单行格式，grep 友好；main/error core 均使用
- `request_id.go` 提供 `WithRequestId`/`RequestIdFromContext`，`zap.go` 的 `*Ctx` 方法自动从 context 提取 requestId 附加字段

## 文件结构

| 文件 | 职责 |
|------|------|
| `config.go` | Config 结构体（含 ErrorFile 错误日志前缀）和默认配置 |
| `writer.go` | LogWriter：文件轮转、日期/大小双维度、旧文件清理 |
| `logger.go` | Init/Get/Close，全局 zap.Logger 管理（atomic.Value + sync.Once 幂等） |
| `field.go` | zap 字段构造函数的包级导出 |
| `zap.go` | Logger 封装类型，常用日志方法 + 6 个带 ctx 的 `*Ctx` 方法 |
| `encoder.go` | 自定义 key=value 编码器，输出 时间 级别 消息 key=value 格式 |
| `request_id.go` | RequestId 上下文注入与读取，供 Ctx 日志方法自动附加 |

## Lobby 项目使用方式

主要消费者是 `lobby` 项目（`D:/self/go/web/Lobby/`），通过 `go.mod` 的 `replace` 指令引用本地路径。
Lobby 代码中**零 zap 直接引用**，全部通过 logger 包操作。

### 初始化模式

`main.go` 中一次性初始化，丢弃 `*zap.Logger` 返回值，改用 `logger.L()` 获取 `*logger.Logger`：

```go
// main.go
import "github.com/go-meridian/logger"

// 初始化（丢弃 *zap.Logger 返回值）
_, err := logger.Init(logCfg)
if err != nil {
    panic("logger.Init error: " + err.Error())
}
defer logger.Close()

// 获取全局 Logger 包装实例
log := logger.L()

// 传递 *logger.Logger 给各模块
handler.Init(log)
httpHandler.Init(log)
natsHandler.Init(mqClient, log)
service.Init(log, natsHandler.GetPublisher())
```

### 模块接收方式

各模块统一接收 `*logger.Logger` 类型，存为包级变量：

```go
// handler/router.go、service/init.go、handler/nats/init.go 等
var log *logger.Logger

func Init(l *logger.Logger) {
    log = l
}
```

结构体中也持有 `*logger.Logger`：

```go
type APIHandler struct {
    Logger *logger.Logger
}
```

### 日志记录方式

统一使用 `logger` 包的字段构造函数，不引入 zap：

```go
// 通过包级变量 log（大多数模块）
log.Info("RouteCmd", logger.String("cmd", cmd), logger.Uint64("uid", uid))
log.Error("panic recovered", logger.String("requestID", requestID), logger.Error(err))
log.Fatal("db.Init error", logger.String("error", ce.Error()))

// 通过 logger.L() 直接调用（无包级 log 变量的场景）
logger.L().Info("PingService", logger.String("requestID", requestID), logger.Uint64("uid", uid))
```

### 实际使用的字段构造函数

Lobby 中使用的字段构造函数（按频率排序）：

| 函数 | 用途 |
|------|------|
| `logger.String(key, val)` | 最常用，字符串字段 |
| `logger.Uint64(key, val)` | 用户 ID、会话 ID 等 |
| `logger.Error(err)` | 错误字段（key 固定为 "error"） |
| `logger.Int(key, val)` | 整数字段（如 dataLen） |
| `logger.Int64(key, val)` | 大整数字段（如 latency_ms） |

### 配置映射

```yaml
# Lobby/config.yaml
log:
  level: "info"
  logFile: "lobby"
  maxSize: 500
  maxAge: 30
```

- `LogDir` 字段**不在配置文件中**，Lobby 在代码中硬编码为 `"logs"`
- 配置加载到 Lobby 自己的 `config.LogConfig`（无 LogDir 字段），再手动构造 `logger.Config`
- `ErrorFile`（`errorFile`）为可选字段：非空时 error 级别日志独立写入该前缀文件，为空则不单独输出

### 注意事项

- `Init()` 返回的 `*zap.Logger` 被 Lobby 丢弃，统一用 `logger.L()` 获取 `*logger.Logger`
- **不用** `logger.Get()` — 仅 mq 库内部 vendor 依赖使用，Lobby 自身不调用
- **不用** `InitWithDefault()` — 始终使用 `Init()` + 显式 Config
- 旧的 `common/logger` 包已被完全移除，Lobby 中无任何引用
- 修改 Config 结构体时需考虑向后兼容，Lobby 不一定使用所有字段
- `field.go` 中新增字段构造函数时需保持与 zap 同名、同签名，方便消费方迁移
- `mq` 库（`github.com/go-meridian/mq`）通过 `logger.Get()` 获取 `*zap.Logger`，修改 `Get()` 签名需同步更新该库
