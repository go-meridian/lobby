package logger

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"time"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// consoleTimeLayout 日志时间格式，与 zap 的 ISO8601 一致
const consoleTimeLayout = "2006-01-02T15:04:05.000Z0700"

// encoderBufferPool 复用缓冲，减少分配
var encoderBufferPool = buffer.NewPool()

// keyValueEncoder 自定义日志编码器，输出格式：
// 2026-09-23T13:10:36.426+0800 INFO RouteCmd cmd=xxx uid=123 requestId=abc-123
type keyValueEncoder struct {
	buf *buffer.Buffer // 累积 With 添加的字段
}

func newKeyValueEncoder() *keyValueEncoder {
	return &keyValueEncoder{buf: encoderBufferPool.Get()}
}

// Clone 复制编码器，同时复制已累积的字段上下文
func (e *keyValueEncoder) Clone() zapcore.Encoder {
	clone := newKeyValueEncoder()
	clone.buf.AppendBytes(e.buf.Bytes())
	return clone
}

// EncodeEntry 编码单条日志为 时间 级别 消息 key=value 格式
func (e *keyValueEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := encoderBufferPool.Get()
	line.AppendTime(ent.Time, consoleTimeLayout)
	line.AppendString(" ")
	line.AppendString(ent.Level.CapitalString())
	line.AppendString(" ")
	line.AppendString(ent.Message)

	if e.buf.Len() > 0 {
		line.AppendByte(' ')
		line.AppendBytes(e.buf.Bytes())
	}

	if len(fields) > 0 {
		tmp := newKeyValueEncoder()
		for i := range fields {
			fields[i].AddTo(tmp)
		}
		if tmp.buf.Len() > 0 {
			line.AppendByte(' ')
			line.AppendBytes(tmp.buf.Bytes())
		}
		tmp.buf.Free()
	}

	line.AppendByte('\n')
	return line, nil
}

// sep 在字段之间追加空格分隔符
func (e *keyValueEncoder) sep() {
	if e.buf.Len() > 0 {
		e.buf.AppendByte(' ')
	}
}

// addKeyValue 追加一个 key=value 字段
func (e *keyValueEncoder) addKeyValue(key, value string) {
	e.sep()
	e.buf.AppendString(key)
	e.buf.AppendByte('=')
	e.appendValue(value)
}

// appendValue 追加字段值，含空白或引号时用引号包裹并转义
func (e *keyValueEncoder) appendValue(v string) {
	if needsQuote(v) {
		e.buf.AppendString(strconv.Quote(v))
	} else {
		e.buf.AppendString(v)
	}
}

// needsQuote 判断值是否需要引号包裹
func needsQuote(v string) bool {
	if v == "" {
		return true
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		if c <= ' ' || c == '"' || c == '\\' {
			return true
		}
	}
	return false
}

// ==================== 基础类型字段 ====================

// AddString 追加字符串字段
func (e *keyValueEncoder) AddString(key, val string) {
	e.addKeyValue(key, val)
}

// AddBool 追加布尔字段
func (e *keyValueEncoder) AddBool(key string, val bool) {
	e.addKeyValue(key, strconv.FormatBool(val))
}

// AddInt64 追加 int64 字段
func (e *keyValueEncoder) AddInt64(key string, val int64) {
	e.addKeyValue(key, strconv.FormatInt(val, 10))
}

// AddInt 追加 int 字段
func (e *keyValueEncoder) AddInt(k string, v int) { e.AddInt64(k, int64(v)) }

// AddInt32 追加 int32 字段
func (e *keyValueEncoder) AddInt32(k string, v int32) { e.AddInt64(k, int64(v)) }

// AddInt16 追加 int16 字段
func (e *keyValueEncoder) AddInt16(k string, v int16) { e.AddInt64(k, int64(v)) }

// AddInt8 追加 int8 字段
func (e *keyValueEncoder) AddInt8(k string, v int8) { e.AddInt64(k, int64(v)) }

// AddUint64 追加 uint64 字段
func (e *keyValueEncoder) AddUint64(key string, val uint64) {
	e.addKeyValue(key, strconv.FormatUint(val, 10))
}

// AddUint 追加 uint 字段
func (e *keyValueEncoder) AddUint(k string, v uint) { e.AddUint64(k, uint64(v)) }

// AddUint32 追加 uint32 字段
func (e *keyValueEncoder) AddUint32(k string, v uint32) { e.AddUint64(k, uint64(v)) }

// AddUint16 追加 uint16 字段
func (e *keyValueEncoder) AddUint16(k string, v uint16) { e.AddUint64(k, uint64(v)) }

// AddUint8 追加 uint8 字段
func (e *keyValueEncoder) AddUint8(k string, v uint8) { e.AddUint64(k, uint64(v)) }

// AddUintptr 追加 uintptr 字段
func (e *keyValueEncoder) AddUintptr(k string, v uintptr) { e.AddUint64(k, uint64(v)) }

// AddFloat64 追加 float64 字段
func (e *keyValueEncoder) AddFloat64(key string, val float64) {
	e.addKeyValue(key, strconv.FormatFloat(val, 'f', -1, 64))
}

// AddFloat32 追加 float32 字段
func (e *keyValueEncoder) AddFloat32(key string, val float32) {
	e.addKeyValue(key, strconv.FormatFloat(float64(val), 'f', -1, 32))
}

// AddComplex128 追加 complex128 字段
func (e *keyValueEncoder) AddComplex128(key string, val complex128) {
	e.addKeyValue(key, strconv.FormatComplex(val, 'f', -1, 128))
}

// AddComplex64 追加 complex64 字段
func (e *keyValueEncoder) AddComplex64(key string, val complex64) {
	e.addKeyValue(key, strconv.FormatComplex(complex128(val), 'f', -1, 64))
}

// AddDuration 追加时长字段
func (e *keyValueEncoder) AddDuration(key string, val time.Duration) {
	e.addKeyValue(key, val.String())
}

// AddTime 追加时间字段
func (e *keyValueEncoder) AddTime(key string, val time.Time) {
	e.addKeyValue(key, val.Format(consoleTimeLayout))
}

// AddBinary 追加二进制字段（base64 编码）
func (e *keyValueEncoder) AddBinary(key string, val []byte) {
	e.addKeyValue(key, base64.StdEncoding.EncodeToString(val))
}

// AddByteString 追加 UTF-8 字节切片字段
func (e *keyValueEncoder) AddByteString(key string, val []byte) {
	e.addKeyValue(key, string(val))
}

// ==================== 复合类型字段（退化为 JSON 值） ====================

// AddReflected 追加反射字段
func (e *keyValueEncoder) AddReflected(key string, val interface{}) error {
	return e.addJSONValue(key, val)
}

// AddObject 追加对象字段
func (e *keyValueEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	return e.addJSONValue(key, obj)
}

// AddArray 追加数组字段
func (e *keyValueEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	return e.addJSONValue(key, arr)
}

// addJSONValue 以 JSON 形式追加复合类型字段值
func (e *keyValueEncoder) addJSONValue(key string, val interface{}) error {
	b, err := json.Marshal(val)
	if err != nil {
		return err
	}
	e.addKeyValue(key, string(b))
	return nil
}

// OpenNamespace 命名空间在 key=value 格式下忽略，内部字段平铺到顶层
func (e *keyValueEncoder) OpenNamespace(key string) {}
