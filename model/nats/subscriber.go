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

	natsLib "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

var coreReqIDCounter uint64

// CoreSubscriber NATS Core 订阅处理器
type CoreSubscriber struct {
	client       NATSClient
	subject      string
	handler      model.HandlerFunc
	workerCount  int
	logger       *zap.Logger
	jobChan      chan *natsLib.Msg
	wg           sync.WaitGroup
	subscription *natsLib.Subscription
}

// NewCoreSubscriber 创建 Core 订阅处理器
func NewCoreSubscriber(client NATSClient, subject string, handler model.HandlerFunc, workerCount int, logger *zap.Logger) *CoreSubscriber {
	if workerCount <= 0 {
		workerCount = runtime.NumCPU() * 2
	}
	return &CoreSubscriber{
		client:      client,
		subject:     subject,
		handler:     handler,
		workerCount: workerCount,
		logger:      logger,
		jobChan:     make(chan *natsLib.Msg, workerCount*2),
	}
}

// Start 启动订阅和 Worker 池
func (s *CoreSubscriber) Start() error {
	// 启动 Worker 池
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	s.logger.Info("Core subscriber worker pool started", zap.Int("workers", s.workerCount))

	// 订阅 NATS Core 主题
	sub, err := s.client.Subscribe(s.subject, func(msg *natsLib.Msg) {
		s.jobChan <- msg
	})
	if err != nil {
		return fmt.Errorf("subscribe error: %w", err)
	}
	s.subscription = sub

	s.logger.Info("Core subscriber started",
		zap.String("subject", s.subject),
		zap.Int("workers", s.workerCount),
	)
	return nil
}

// Stop 停止订阅和 Worker 池
func (s *CoreSubscriber) Stop() {
	if s.subscription != nil {
		s.subscription.Unsubscribe()
	}
	close(s.jobChan)
	s.wg.Wait()
	s.logger.Info("Core subscriber stopped")
}

// worker Worker 协程
func (s *CoreSubscriber) worker(id int) {
	defer s.wg.Done()
	for msg := range s.jobChan {
		s.processMsg(id, msg)
	}
}

// processMsg 处理单条消息
func (s *CoreSubscriber) processMsg(workerID int, msg *natsLib.Msg) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Core subscriber worker panic recovered",
				zap.Int("worker", workerID),
				zap.Any("panic", r),
			)
		}
	}()

	req := &gateway.Request{}
	if err := json.Unmarshal(msg.Data, req); err != nil {
		s.logger.Error("Unmarshal request failed",
			zap.Int("worker", workerID),
			zap.Error(err),
		)
		s.replyError(msg, "", -1, fmt.Sprintf("invalid request: %s", err.Error()))
		return
	}

	requestID := req.RequestID
	if requestID == "" {
		requestID = "core_" + strconv.FormatUint(atomic.AddUint64(&coreReqIDCounter, 1), 10) + "_" + strconv.FormatInt(time.Now().UnixMilli()%100000, 10)
	}

	s.logger.Info("Processing core request",
		zap.String("requestID", requestID),
		zap.Int("worker", workerID),
		zap.String("cmd", req.Cmd),
		zap.Uint64("uid", req.UID),
	)

	result, ce := s.handler(requestID, req.Cmd, req.UID, req.Data)
	if ce != nil {
		s.logger.Error("Core handler failed",
			zap.String("requestID", requestID),
			zap.Int("worker", workerID),
			zap.String("cmd", req.Cmd),
			zap.String("error", ce.Error()),
		)
		s.replyError(msg, requestID, int(ce.GetCode()), ce.GetMsg())
		return
	}

	s.replySuccess(msg, requestID, result)
}

// replySuccess 回复成功响应
func (s *CoreSubscriber) replySuccess(msg *natsLib.Msg, requestID string, data interface{}) {
	rsp := &gateway.Response{
		RequestID: requestID,
		Code:      0,
		Message:   "success",
		Data:      data,
	}
	rspData, _ := json.Marshal(rsp)
	if err := msg.Respond(rspData); err != nil {
		s.logger.Error("Core respond failed", zap.Error(err))
	}
}

// replyError 回复错误响应
func (s *CoreSubscriber) replyError(msg *natsLib.Msg, requestID string, code int, message string) {
	rsp := &gateway.Response{
		RequestID: requestID,
		Code:      code,
		Message:   message,
	}
	rspData, _ := json.Marshal(rsp)
	if err := msg.Respond(rspData); err != nil {
		s.logger.Error("Core respond failed", zap.Error(err))
	}
}
