package mqhandler

import (
	"strconv"
	"sync/atomic"

	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/lobby/model/proto/common"
	"github.com/go-meridian/lobby/model/proto/gate"
	"github.com/go-meridian/logger"
	"github.com/go-meridian/mq"
	"google.golang.org/protobuf/proto"
)

var (
	client    mq.MQClient
	log       *logger.Logger
	publisher *mqPublisher
)

// mqPublisher MQ 发布器封装
type mqPublisher struct {
	client  mq.MQClient
	subject string
}

// Publish 发送 GatePush 到指定 connId
func (p *mqPublisher) Publish(connId uint64, msgId uint32, payload []byte) error {
	push := &gate.GatePush{
		ConnId:  connId,
		MsgId:   msgId,
		Payload: payload,
	}
	data, err := proto.Marshal(push)
	if err != nil {
		return err
	}

	subject := p.subject + "." + strconv.FormatUint(uint64(msgId), 10)
	ce := p.client.Publish(subject, data)
	if ce != nil {
		return ce
	}
	return nil
}

// PublishBroadcast 广播消息（connId=0）
func (p *mqPublisher) PublishBroadcast(msgId uint32, payload []byte) error {
	return p.Publish(0, msgId, payload)
}

// MQHandler MQ 消息处理函数签名（含 msgId 用于路由）
type MQHandler func(connId uint64, requestId uint64, msgId uint32, payload []byte) ([]byte, *codeerror.CodeError)

// coreSubscriptionEntry Core NATS 订阅注册
type coreSubscriptionEntry struct {
	subject     string
	handler     MQHandler
	workerCount int
}

// coreSubscriptionRegistry Core NATS 订阅注册表
var coreSubscriptionRegistry []coreSubscriptionEntry

// publishStreamEntry 发送端 Stream 注册
type publishStreamEntry struct {
	streamName    string
	streamSubject string
}

// publishStreamRegistry 发送端 Stream 注册表
var publishStreamRegistry []publishStreamEntry

// natsReqIDCounter NATS 请求 ID 计数器
var natsReqIDCounter uint64

// Init 初始化 MQ handler 层
func Init(c mq.MQClient, l *logger.Logger) {
	client = c
	log = l
}

// RegisterCoreSubscription 注册 MQ Core 订阅
func RegisterCoreSubscription(subject string, h MQHandler, workerCount int) {
	coreSubscriptionRegistry = append(coreSubscriptionRegistry, coreSubscriptionEntry{
		subject:     subject,
		handler:     h,
		workerCount: workerCount,
	})
}

// RegisterPublishStream 注册 MQ 发送 Stream
func RegisterPublishStream(streamName, streamSubject string) {
	publishStreamRegistry = append(publishStreamRegistry, publishStreamEntry{
		streamName:    streamName,
		streamSubject: streamSubject,
	})
}

// GetPublisher 获取发布器
func GetPublisher() *mqPublisher {
	return publisher
}

// IsConnected 检查 MQ 连接状态
func IsConnected() bool {
	if client == nil {
		return false
	}
	return client.IsConnected()
}

// GetRegisteredSubscriptions 获取已注册的订阅信息（用于启动日志）
func GetRegisteredSubscriptions() []string {
	var result []string
	for _, entry := range coreSubscriptionRegistry {
		result = append(result, entry.subject)
	}
	for _, entry := range publishStreamRegistry {
		result = append(result, entry.streamSubject+".{msgId} (publish)")
	}
	return result
}

// Start 启动所有已注册的订阅和发送 Stream
func Start() *codeerror.CodeError {
	for _, entry := range coreSubscriptionRegistry {
		localEntry := entry
		_, ce := client.Subscribe(localEntry.subject, func(msg mq.Message) {
			handleMessage(msg, localEntry.handler)
		})
		if ce != nil {
			return codeerror.SystemError.Msg("MQ subscribe error: " + ce.Error())
		}
		log.Info("MQ subscription registered",
			logger.String("subject", localEntry.subject),
			logger.Int("workers", localEntry.workerCount),
		)
	}

	if len(publishStreamRegistry) > 0 {
		ps := publishStreamRegistry[0]
		publisher = &mqPublisher{
			client:  client,
			subject: ps.streamSubject,
		}
		log.Info("MQ publish stream registered",
			logger.String("stream", ps.streamName),
			logger.String("subject", ps.streamSubject+".{msgId}"),
		)
	}

	return nil
}

// generateNatsRequestID 生成 NATS 请求 ID
func generateNatsRequestID() uint64 {
	return atomic.AddUint64(&natsReqIDCounter, 1)
}

// handleMessage 解析 Protobuf 消息并路由到对应 handler
func handleMessage(msg mq.Message, routeHandler MQHandler) {
	// 解析 GateRequest protobuf
	var req gate.GateRequest
	if err := proto.Unmarshal(msg.Data(), &req); err != nil {
		log.Error("MQ GateRequest unmarshal error",
			logger.String("subject", msg.Subject()),
			logger.Error(err),
		)
		return
	}

	requestId := req.RequestId
	if requestId == 0 {
		requestId = generateNatsRequestID()
	}

	// 路由到对应 handler
	payload, ce := routeHandler(req.ConnId, requestId, req.MsgId, req.Payload)

	// 构造 GateResponse
	resp := &gate.GateResponse{
		ConnId:    req.ConnId,
		RequestId: requestId,
		MsgId:     req.MsgId,
	}

	if ce != nil {
		// 构造错误的 ErrMsg 放入 payload
		errMsg := &common.ErrMsg{
			RetCode: int32(ce.GetCode()),
			ErrMsg:  ce.Error(),
		}
		errBytes, _ := proto.Marshal(errMsg)
		resp.Payload = errBytes
	} else {
		resp.Payload = payload
	}

	// 回写响应（Request/Reply 模式）
	replyTo := msg.ReplyTo()
	if replyTo == "" {
		return
	}

	respBytes, err := proto.Marshal(resp)
	if err != nil {
		log.Error("MQ GateResponse marshal error",
			logger.Uint64("requestId", requestId),
			logger.Error(err),
		)
		return
	}

	if ce := client.Publish(replyTo, respBytes); ce != nil {
		log.Error("MQ response publish error",
			logger.Uint64("requestId", requestId),
			logger.String("replyTo", replyTo),
			logger.String("error", ce.Error()),
		)
	}
}

// MsgHandler 消息处理函数签名（各 service 通过 init 自注册）
type MsgHandler func(connId uint64, requestId uint64, payload []byte) ([]byte, *codeerror.CodeError)

var msgRegistry = make(map[uint32]MsgHandler)

// Register 注册 MsgId 对应的处理函数
func Register(msgId uint32, fn MsgHandler) {
	msgRegistry[msgId] = fn
}

// RouteMsg 根据 MsgId 分发到对应 handler
func RouteMsg(connId uint64, requestId uint64, msgId uint32, payload []byte) ([]byte, *codeerror.CodeError) {
	log.Info("RouteMsg",
		logger.Uint64("connId", connId),
		logger.Uint64("requestId", requestId),
		logger.Uint32("msgId", msgId),
	)

	fn, ok := msgRegistry[msgId]
	if !ok {
		return nil, codeerror.UnknownCmd.Msg("unknown msgId")
	}
	return fn(connId, requestId, payload)
}
