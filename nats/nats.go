package nats

import (
	"lobby/config"
	codeerror2 "lobby/model/codeerror"

	natsClient "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Client NATS 客户端封装
type Client struct {
	Conn   *natsClient.Conn
	Config *config.NATSConfig
	logger *zap.Logger
}

// Init 初始化 NATS 连接
func Init(cfg *config.NATSConfig, logger *zap.Logger) (*Client, *codeerror2.CodeError) {
	nc, err := natsClient.Connect(cfg.URL)
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
