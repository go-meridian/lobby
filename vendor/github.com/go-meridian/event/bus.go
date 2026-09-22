package event

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/go-meridian/job"
)

// BusSubscriber 定义订阅相关行为
type BusSubscriber interface {
	// Subscribe 同步订阅，回调在发布者 goroutine 中直接执行
	Subscribe(topic string, fn interface{}) error
	// SubscribeAsync 异步订阅，回调通过 jobmgr 异步执行
	// transactional 为 true 时，同一 topic 的回调串行执行
	SubscribeAsync(topic string, fn interface{}, transactional bool) error
	// SubscribeOnce 一次性同步订阅，执行后自动移除
	SubscribeOnce(topic string, fn interface{}) error
	// SubscribeOnceAsync 一次性异步订阅，执行后自动移除
	SubscribeOnceAsync(topic string, fn interface{}) error
	// Unsubscribe 取消订阅
	Unsubscribe(topic string, handler interface{}) error
}

// BusPublisher 定义发布相关行为
type BusPublisher interface {
	// Publish 发布事件，触发该 topic 下所有已注册的回调
	Publish(topic string, args ...interface{})
}

// BusController 定义总线控制行为
type BusController interface {
	// HasCallback 检查某 topic 是否有已注册的回调
	HasCallback(topic string) bool
	// WaitAsync 阻塞等待所有异步回调执行完毕
	WaitAsync()
}

// Bus 事件总线接口，组合订阅、发布、控制三类行为
type Bus interface {
	BusSubscriber
	BusPublisher
	BusController
}

// eventHandler 单个事件回调处理器
type eventHandler struct {
	callBack      reflect.Value // 回调函数
	flagOnce      bool          // 是否一次性
	async         bool          // 是否异步
	transactional bool          // 异步模式下是否串行执行
	sync.Mutex                  // 串行执行时的互斥锁
}

// EventBus 事件总线实现
type EventBus struct {
	handlers map[string][]*eventHandler
	lock     sync.Mutex
	wg       sync.WaitGroup
	mgr      *job.JobManager
}

// New 创建一个新的 EventBus 实例，mgr 为 nil 时使用 job.Mgr() 默认实例
func New(mgr *job.JobManager) Bus {
	if mgr == nil {
		mgr = job.Mgr()
	}
	return Bus(&EventBus{
		handlers: make(map[string][]*eventHandler),
		mgr:      mgr,
	})
}

// doSubscribe 内部订阅逻辑，校验回调类型后追加到 handlers
func (bus *EventBus) doSubscribe(topic string, fn interface{}, handler *eventHandler) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return fmt.Errorf("eventbus: %s 不是函数类型", reflect.TypeOf(fn).Kind())
	}
	bus.handlers[topic] = append(bus.handlers[topic], handler)
	return nil
}

// Subscribe 同步订阅
func (bus *EventBus) Subscribe(topic string, fn interface{}) error {
	return bus.doSubscribe(topic, fn, &eventHandler{
		callBack: reflect.ValueOf(fn),
	})
}

// SubscribeAsync 异步订阅，transactional 控制是否串行执行
func (bus *EventBus) SubscribeAsync(topic string, fn interface{}, transactional bool) error {
	return bus.doSubscribe(topic, fn, &eventHandler{
		callBack:      reflect.ValueOf(fn),
		async:         true,
		transactional: transactional,
	})
}

// SubscribeOnce 一次性同步订阅
func (bus *EventBus) SubscribeOnce(topic string, fn interface{}) error {
	return bus.doSubscribe(topic, fn, &eventHandler{
		callBack: reflect.ValueOf(fn),
		flagOnce: true,
	})
}

// SubscribeOnceAsync 一次性异步订阅
func (bus *EventBus) SubscribeOnceAsync(topic string, fn interface{}) error {
	return bus.doSubscribe(topic, fn, &eventHandler{
		callBack: reflect.ValueOf(fn),
		flagOnce: true,
		async:    true,
	})
}

// HasCallback 检查某 topic 是否有已注册的回调
func (bus *EventBus) HasCallback(topic string) bool {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	if handlers, ok := bus.handlers[topic]; ok {
		return len(handlers) > 0
	}
	return false
}

// Unsubscribe 取消订阅
func (bus *EventBus) Unsubscribe(topic string, handler interface{}) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	if _, ok := bus.handlers[topic]; ok && len(bus.handlers[topic]) > 0 {
		bus.removeHandler(topic, bus.findHandlerIdx(topic, reflect.ValueOf(handler)))
		return nil
	}
	return fmt.Errorf("eventbus: topic %s 不存在", topic)
}

// Publish 发布事件，触发该 topic 下所有已注册的回调
func (bus *EventBus) Publish(topic string, args ...interface{}) {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	handlers, ok := bus.handlers[topic]
	if !ok || len(handlers) == 0 {
		return
	}
	// 拷贝 handlers 切片，避免迭代中被修改
	copyHandlers := make([]*eventHandler, len(handlers))
	copy(copyHandlers, handlers)
	for i, handler := range copyHandlers {
		if handler.flagOnce {
			bus.removeHandler(topic, i)
		}
		if !handler.async {
			bus.doPublish(handler, topic, args...)
		} else {
			bus.wg.Add(1)
			if handler.transactional {
				bus.lock.Unlock()
				handler.Lock()
				bus.lock.Lock()
			}
			bus.mgr.AddJob(func() {
				bus.doPublishAsync(handler, topic, args...)
			})
		}
	}
}

// doPublish 同步执行回调
func (bus *EventBus) doPublish(handler *eventHandler, topic string, args ...interface{}) {
	passedArguments := bus.setUpPublish(handler, args...)
	handler.callBack.Call(passedArguments)
}

// doPublishAsync 异步执行回调，通过 jobmgr 投递任务
func (bus *EventBus) doPublishAsync(handler *eventHandler, topic string, args ...interface{}) {
	defer bus.wg.Done()
	if handler.transactional {
		defer handler.Unlock()
	}
	bus.doPublish(handler, topic, args...)
}

// removeHandler 从 handlers 切片中移除指定索引的 handler
func (bus *EventBus) removeHandler(topic string, idx int) {
	if _, ok := bus.handlers[topic]; !ok {
		return
	}
	l := len(bus.handlers[topic])
	if !(0 <= idx && idx < l) {
		return
	}
	copy(bus.handlers[topic][idx:], bus.handlers[topic][idx+1:])
	bus.handlers[topic][l-1] = nil
	bus.handlers[topic] = bus.handlers[topic][:l-1]
}

// findHandlerIdx 根据回调类型和指针查找 handler 索引
func (bus *EventBus) findHandlerIdx(topic string, callback reflect.Value) int {
	if _, ok := bus.handlers[topic]; ok {
		for idx, handler := range bus.handlers[topic] {
			if handler.callBack.Type() == callback.Type() &&
				handler.callBack.Pointer() == callback.Pointer() {
				return idx
			}
		}
	}
	return -1
}

// setUpPublish 构造回调参数，处理 nil 参数的反射创建
func (bus *EventBus) setUpPublish(callback *eventHandler, args ...interface{}) []reflect.Value {
	funcType := callback.callBack.Type()
	passedArguments := make([]reflect.Value, len(args))
	for i, v := range args {
		if v == nil {
			passedArguments[i] = reflect.New(funcType.In(i)).Elem()
		} else {
			passedArguments[i] = reflect.ValueOf(v)
		}
	}
	return passedArguments
}

// WaitAsync 阻塞等待所有异步回调执行完毕
func (bus *EventBus) WaitAsync() {
	bus.wg.Wait()
}
