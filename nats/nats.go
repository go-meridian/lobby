package nats

import (
	"lobby/config"
	codeerror2 "lobby/model/codeerror"

	natsLib "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Client NATS 客户端封装
type Client struct {
	Conn   *natsLib.Conn
	Config *config.NATSConfig
	logger *zap.Logger
}

// Init 初始化 NATS 连接
func Init(cfg *config.NATSConfig, logger *zap.Logger) (*Client, *codeerror2.CodeError) {
	nc, err := natsLib.Connect(cfg.URL)
	if err != nil {
		return nil, codeerror2.NATSError.Msg("nats.Connect error: " + err.Error())
	}

	logger.Info("NATS connected", zap.String("url", cfg.URL))

	return &Client{
		Conn:   nc,
		Config: cfg,
		logger: logger,
	}, nil
}

// Close 关闭 NATS 连接
func (c *Client) Close() {
	if c.Conn != nil {
		c.Conn.Close()
		c.logger.Info("NATS connection closed")
	}
}

// IsConnected 检查连接状态
func (c *Client) IsConnected() bool {
	return c.Conn != nil && c.Conn.IsConnected()
}

// Subscribe 订阅 NATS Core 主题
func (c *Client) Subscribe(subject string, handler natsLib.MsgHandler) (*natsLib.Subscription, error) {
	return c.Conn.Subscribe(subject, handler)
}

// PublishSync 同步发布消息到 NATS Core（不等待回复）
func (c *Client) PublishSync(subject string, data []byte) *codeerror2.CodeError {
	if err := c.Conn.Publish(subject, data); err != nil {
		return codeerror2.NATSError.Msg("publish error: " + err.Error())
	}
	return nil
}
