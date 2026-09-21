package mq

import (
	"runtime"
	"sync"

	"github.com/go-meridian/logger"
)

// HandlerFunc 消息处理函数签名
type HandlerFunc func(msg Message)

// WorkerPool 通用 Worker 池
type WorkerPool struct {
	workerCount int
	jobChan     chan Message
	handler     HandlerFunc
	logger      *logger.Logger
	wg          sync.WaitGroup
}

// NewWorkerPool 创建 WorkerPool
func NewWorkerPool(workerCount int, handler HandlerFunc) *WorkerPool {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU() * 2
	}
	return &WorkerPool{
		workerCount: workerCount,
		jobChan:     make(chan Message, workerCount*2),
		handler:     handler,
		logger:      GetLogger(),
	}
}

// Start 启动所有 Worker
func (p *WorkerPool) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.logger.Info("worker pool started", logger.Int("workers", p.workerCount))
}

// Stop 停止所有 Worker
func (p *WorkerPool) Stop() {
	close(p.jobChan)
	p.wg.Wait()
	p.logger.Info("worker pool stopped")
}

// Submit 提交消息到队列
func (p *WorkerPool) Submit(msg Message) {
	p.jobChan <- msg
}

func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	for msg := range p.jobChan {
		p.processMsg(id, msg)
	}
}

func (p *WorkerPool) processMsg(workerID int, msg Message) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 1024)
			n := runtime.Stack(buf, false)
			p.logger.Error("worker panic recovered",
				logger.Int("worker", workerID),
				logger.Any("panic", r),
				logger.String("stack", string(buf[:n])),
			)
			_ = msg.Nak()
		}
	}()
	p.handler(msg)
}
