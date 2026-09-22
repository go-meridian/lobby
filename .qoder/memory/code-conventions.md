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
- 日志中必须包含 `requestID` 或 `connId` 等链路标识

```go
// 标准模式
package xxx

var log *logger.Logger

func Init(l *logger.Logger) {
    log = l
}
```

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
