# 代码规范

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

## 日志模式

所有包统一使用包级 `var log *logger.Logger`，通过 `Init(l *logger.Logger)` 初始化。

- 禁止将 log 作为函数参数传递
- 禁止在结构体中持有 log 字段
- 禁止 `fmt.Println` 调试
- **所有日志必须使用 `xxCtx` 形式**：`InfoCtx`/`ErrorCtx`/`WarnCtx`/`DebugCtx`/`FatalCtx`/`DPanicCtx`
- **RequestId 必须通过 `logger.WithRequestId(ctx, requestId)` 放入 ctx**，由 `xxCtx` 方法自动附加 `requestId=xxx` 字段，禁止手动传 `requestID` 字段

```go
// 标准模式
package xxx

var log *logger.Logger

func Init(l *logger.Logger) {
    log = l
}

// 请求/消息链路中携带含 requestId 的 ctx
log.InfoCtx(ctx, "RouteMsg", logger.Uint64("connId", connId))
log.ErrorCtx(ctx, "panic recovered", logger.Error(err))

// 无请求上下文的启动/后台日志使用 context.Background()
log.InfoCtx(context.Background(), "server started", logger.String("addr", addr))
```

### RequestId 流转

- **HTTP**：`RequestID` 中间件生成 requestID 后，`logger.WithRequestId(c.Request().Context(), requestID)` 写入请求 ctx，后续中间件和 handler 用 `xxCtx(c.Request().Context(), ...)`
- **NATS**：`handleMessage` 生成 requestId（`req.RequestId` 为空则自动生成 uint64 递增 ID），`logger.WithRequestId(context.Background(), strconv.FormatUint(requestId, 10))` 写入 ctx 后沿 `Handler` → `RouteMsg` → `MsgHandler` 向下传递
- `logger.WithRequestId` 接受 string 类型，uint64 的 requestId 需 `strconv.FormatUint` 转换（转换后仅用于日志，`GateResponse.RequestId` 仍回传原始 uint64）

### 日志输出格式

logger v1.0.4 使用自定义 `keyValueEncoder`，输出单行 `key=value` 格式（grep 友好）：

```
2026-09-23T13:10:36.426+0800 INFO RouteMsg connId=123 msgId=1001 requestId=abc-123
```

- 格式：`时间 大写级别 消息 key=value ...`
- 错误日志可通过 `Config.ErrorFile` 独立输出到单独文件
- 含空白或引号的字段值自动加引号转义

## 常量

放在 `model/constant/` 下，按领域建文件。

## 时间函数

禁止直接调用 `time.Now()` 等时间函数，必须通过 `common/util/utilTime.go` 封装。

## 禁止事项

- 禁止 emoji 出现在代码、注释、文档中
- 禁止 `_ = someFunc()`，所有错误必须显式处理
- 禁止硬编码密钥/Token/密码
- 禁止 `fmt.Println` 调试
- 禁止直接调用 `time.Now()` 等时间函数

## 代码组织

- 小文件优先：宁可多个小文件，不要少数大文件
- 高内聚低耦合：每个文件聚焦单一领域
- 文件大小限制：典型 200-400 行，单文件不超过 800 行
- 按领域组织：按功能/业务域划分目录结构

## 配置

`config.yaml` 通过 viper 加载，结构体定义在 `config/config.go`。
