# AGENTS.md

This file provides guidance to the AI agent when working with code in this repository.

## Project

Go library (`github.com/SilentQianyi/mq`) providing a unified MQ client interface with NATS and Redis backends. Uses factory pattern: sub-packages register via `init()` + `RegisterFactory()`.

## Error Types

All error returns use `*ce.CodeError` (from `github.com/SilentQianyi/codeerror`), NOT Go standard `error`. Package-level error codes are defined in `error.go`. When returning errors, use `.Msg()` to wrap details.

## Logging

本项目日志统一使用 `github.com/SilentQianyi/logger` 包，不直接依赖 `go.uber.org/zap`。
- 获取实例：`mq.GetLogger()` 或 `logger.L()` 返回 `*logger.Logger`
- 日志字段：`logger.String()`、`logger.Int()`、`logger.Error()` 等
- logger 包必须在 mq 初始化前完成 `logger.Init()`
- 禁止使用 `fmt.Println` 或 `log` 包

## Naming Conventions

- Sub-packages (`nats/`, `redis/`) import parent as `mq "github.com/SilentQianyi/mq"` (named import)
- Third-party libs use short aliases: `natsLib`, `ce`

## Lobby 项目使用方式（下游消费者）

Lobby 通过 `replace` 指令引用本库（`replace github.com/SilentQianyi/mq => ../mq`），以下为标准集成模式。

### 1. Import（main.go）

```go
import (
    mq "github.com/SilentQianyi/mq"
    _ "github.com/SilentQianyi/mq/nats"   // side-effect，注册 NATS 工厂
    _ "github.com/SilentQianyi/mq/redis"  // side-effect，注册 Redis 工厂
)
```

必须至少导入一个实现包，否则 `mq.NewClient` 报 "unsupported mq mode"。

### 2. 初始化（main.go）

```go
// logger 包需先初始化（mq 内部依赖 logger.Get()）
logger.Init(cfg)

mqCfg := &mq.Config{
    Mode: mq.ModeNATS,
    NATS: &mq.NATSConfig{URL: cfg.NATS.URL},
}
mqClient, err := mq.NewClient(mqCfg)
defer mqClient.Close()
```

### 3. Handler 层封装（handler/nats/init.go）

Lobby 将 `mq.MQClient` 封装为 `mqPublisher`，通过 `subject + "." + cmd` 做路由：

```go
type mqPublisher struct {
    client  mq.MQClient
    subject string  // 例: "lobby.gate"
}

func (p *mqPublisher) Publish(cmd string, data []byte) error {
    subject := p.subject + "." + cmd  // "lobby.gate.ping"
    ce := p.client.Publish(subject, data)
    if ce != nil {
        return ce
    }
    return nil
}
```

### 4. 自注册模式

Lobby 使用 init() + 全局注册表模式：

- `RegisterCoreSubscription(subject, handler, workerCount)` — Core NATS 订阅
- `RegisterPublishStream(streamName, streamSubject)` — 发布 Stream
- `Register()` — 启动时批量激活所有已注册项

### 5. Subscribe 使用

```go
client.Subscribe(entry.subject, func(msg mq.Message) {
    // 处理消息
    data := msg.Data()
    // msg.Ack() / msg.Nak() 用于队列模式
})
```

### 6. 已知限制

mq.Message 接口当前不支持 `Respond()` 方法。Lobby 的 Request/Reply 模式通过原生 `*nats.Msg.Respond()` 实现（`model/nats/` 层），不经过 mq 抽象层。
