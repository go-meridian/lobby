# AGENTS.md

## 项目概述

Go 库包，提供基于发布-订阅模式的事件总线。
模块: `github.com/go-meridian/event` -- 包名 `eventbus`。

## 核心设计

- 基于反射的动态回调派发
- 接口分离: `BusSubscriber` / `BusPublisher` / `BusController` / `Bus`
- `Event` 接口要求实现 `Topic() string`，返回事件主题
- 异步回调通过 `jobmgr.Mgr().AddJob()` 投递到 goroutine 池
- 全局单例: `Init()` + `Default()`，支持自动初始化

## 依赖

- `github.com/go-meridian/job` (jobmgr)，本地 replace `../job`

## 文件结构

```
bus.go       核心接口 + EventBus 实现
global.go    全局单例 + Event 接口 + 包级便捷函数
```

## 核心 API

### 接口

```go
type Bus interface {
    Subscribe(topic string, fn interface{}) error
    SubscribeAsync(topic string, fn interface{}, transactional bool) error
    SubscribeOnce(topic string, fn interface{}) error
    SubscribeOnceAsync(topic string, fn interface{}) error
    Unsubscribe(topic string, handler interface{}) error
    Publish(topic string, args ...interface{})
    HasCallback(topic string) bool
    WaitAsync()
}

type Event interface {
    Topic() string
}
```

### 全局便捷函数

```go
func Init()                                      // 显式初始化
func Default() Bus                               // 获取全局实例（自动初始化）
func SubscribeAsync(event Event, fn interface{})  // 异步订阅（transactional=true）
func Publish(event Event)                        // 异步发布（通过 jobmgr）
```

## 使用示例

### 定义事件

```go
type EventUpdateGroup struct {
    RoomId  string
    GroupId int32
}

func (e *EventUpdateGroup) Topic() string {
    return "lobby:update:group"
}
```

### 订阅事件

```go
eventbus.Init()

eventbus.SubscribeAsync((*EventUpdateGroup)(nil), func(event *EventUpdateGroup) {
    // 处理分组更新
})
```

### 发布事件

```go
eventbus.Publish(&EventUpdateGroup{
    RoomId:  "room_001",
    GroupId: 1,
})
```

## 代码约定

- 所有注释使用中文
- 回调函数通过反射调用，签名必须匹配事件类型
- SubscribeAsync 的 transactional=true 保证同一 topic 的回调串行执行
- Publish 通过 jobmgr 异步执行，不阻塞调用方

## 构建验证

```bash
go build ./...
go vet ./...
```
