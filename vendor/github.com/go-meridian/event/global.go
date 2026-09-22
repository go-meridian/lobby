package event

import (
	"sync"

	"github.com/go-meridian/job"
)

// Event 事件接口，所有事件结构体必须实现 Topic 方法
type Event interface {
	Topic() string
}

var (
	globalBus Bus
	globalMgr *job.JobManager
	once      sync.Once
)

// Init 显式初始化全局事件总线，mgr 为 nil 时使用 job.Mgr() 默认实例
func Init(mgr *job.JobManager) {
	once.Do(func() {
		if mgr != nil {
			globalMgr = mgr
		} else {
			globalMgr = job.Mgr()
		}
		globalBus = New(globalMgr)
	})
}

// Default 获取全局事件总线实例，未初始化时自动创建
func Default() Bus {
	once.Do(func() {
		globalMgr = job.Mgr()
		globalBus = New(globalMgr)
	})
	return globalBus
}

// SubscribeAsync 异步订阅事件，transactional=true 保证同一 topic 串行执行
func SubscribeAsync(event Event, fn interface{}) {
	Default().SubscribeAsync(event.Topic(), fn, true)
}

// Publish 异步发布事件，通过 jobmgr 投递到 goroutine 池执行
func Publish(event Event) {
	Default() // 确保已初始化
	globalMgr.AddJob(func() {
		globalBus.Publish(event.Topic(), event)
	})
}
