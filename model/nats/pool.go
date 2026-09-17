package natsModel

import (
	"encoding/json"
	"fmt"
	"lobby/model"
	"lobby/model/gateway"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

var reqIDCounter uint64

// WorkerPool Worker 池
type WorkerPool struct {
	workerCount int
	jobChan     chan *nats.Msg
	handler     model.HandlerFunc
	logger      *zap.Logger
	wg          sync.WaitGroup
}

// NewWorkerPool 创建 Worker 池
func NewWorkerPool(workerCount int, handler model.HandlerFunc, logger *zap.Logger) *WorkerPool {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU() * 2
	}
	return &WorkerPool{
		workerCount: workerCount,
		jobChan:     make(chan *nats.Msg, workerCount*2),
		handler:     handler,
		logger:      logger,
	}
}

// Start 启动 Worker 池
func (p *WorkerPool) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.logger.Info("Worker pool started", zap.Int("workers", p.workerCount))
}

// Stop 停止 Worker 池
func (p *WorkerPool) Stop() {
	close(p.jobChan)
	p.wg.Wait()
	p.logger.Info("Worker pool stopped")
}

// Submit 提交消息到 Worker 池
func (p *WorkerPool) Submit(msg *nats.Msg) {
	p.jobChan <- msg
}

// worker Worker 协程
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()
	for msg := range p.jobChan {
		p.processMsg(id, msg)
	}
}

// processMsg 处理单条消息
func (p *WorkerPool) processMsg(workerID int, msg *nats.Msg) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("Worker panic recovered",
				zap.Int("worker", workerID),
				zap.Any("panic", r),
			)
			if err := msg.Nak(); err != nil {
				p.logger.Error("Nak failed", zap.Error(err))
			}
		}
	}()

	req := &gateway.Request{}
	if err := json.Unmarshal(msg.Data, req); err != nil {
		p.logger.Error("Unmarshal request failed",
			zap.Int("worker", workerID),
			zap.Error(err),
		)
		p.replyError(msg, "", -1, fmt.Sprintf("invalid request: %s", err.Error()))
		msg.Ack()
		return
	}

	requestID := req.RequestID
	if requestID == "" {
		requestID = "nats_" + strconv.FormatUint(atomic.AddUint64(&reqIDCounter, 1), 10) + "_" + strconv.FormatInt(time.Now().UnixMilli()%100000, 10)
	}

	p.logger.Info("Processing request",
		zap.String("requestID", requestID),
		zap.Int("worker", workerID),
		zap.String("cmd", req.Cmd),
		zap.Uint64("uid", req.UID),
	)

	result, ce := p.handler(requestID, req.Cmd, req.UID, req.Data)
	if ce != nil {
		p.logger.Error("Handler failed",
			zap.String("requestID", requestID),
			zap.Int("worker", workerID),
			zap.String("cmd", req.Cmd),
			zap.String("error", ce.Error()),
		)
		p.replyError(msg, requestID, int(ce.GetCode()), ce.GetMsg())
		msg.Ack()
		return
	}

	p.replySuccess(msg, requestID, result)
	msg.Ack()
}

// replySuccess 回复成功响应
func (p *WorkerPool) replySuccess(msg *nats.Msg, requestID string, data interface{}) {
	rsp := &gateway.Response{
		RequestID: requestID,
		Code:      0,
		Message:   "success",
		Data:      data,
	}
	rspData, _ := json.Marshal(rsp)
	if err := msg.Respond(rspData); err != nil {
		p.logger.Error("Respond failed", zap.Error(err))
	}
}

// replyError 回复错误响应
func (p *WorkerPool) replyError(msg *nats.Msg, requestID string, code int, message string) {
	rsp := &gateway.Response{
		RequestID: requestID,
		Code:      code,
		Message:   message,
	}
	rspData, _ := json.Marshal(rsp)
	if err := msg.Respond(rspData); err != nil {
		p.logger.Error("Respond failed", zap.Error(err))
	}
}
