# AGENTS.md

This file provides guidance to the AI agent when working with code in this repository.

## Project Overview

Go library package (not a binary). Wraps `go.uber.org/zap` with a custom `LogWriter` that rotates log files by both date and size.

## Conventions

- Package name: `logger` — all files use `package logger`
- Comments and documentation in Chinese
- No `main` package — this is imported by other projects
- No tests exist yet — when adding tests, use `_test.go` files in the same package

## Key Design

- `LogWriter` (writer.go) is the core: mutex-protected, dual rotation (daily date + max file size), auto-cleans files older than `maxAge` days
- File naming: `{prefix}.{YYYYMMDD}.{seq}` (e.g. `app.20260920.1`)
- Global singleton pattern: `Init()` sets a package-level `*zap.Logger`, retrieved via `Get()`

## Lobby 项目使用方式

主要消费者是 `lobby` 项目（`D:/self/go/web/Lobby/`），通过 `go.mod` 的 `replace` 指令引用本地路径。

### 初始化模式

`main.go` 中一次性初始化，返回的 `*zap.Logger` 通过函数参数逐层传递给各模块：

```go
// main.go
zapLog, err := logger.Init(logCfg)
defer logger.Close()

// 传递给各模块，各模块存为包级 var logger *zap.Logger
handler.Init(zapLog)
service.Init(zapLog, publisher)
```

- **不用** `logger.Get()` — Lobby 项目完全通过参数传递 `*zap.Logger`
- **不用** `InitWithDefault()` — 始终使用 `Init()` + 显式 Config
- 各模块（handler、service、nats）各自声明 `var logger *zap.Logger` 作为包级变量

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

### 注意事项

- Lobby 有一个旧的 `common/logger` 包（本地封装），已被本包取代，当前未被使用
- 修改 Config 结构体时需考虑向后兼容，Lobby 不一定使用所有字段
