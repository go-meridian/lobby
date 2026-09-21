package elect

import (
	"github.com/go-meridian/logger"
)

var (
	elector Elector
	log    *logger.Logger
)

// Init 初始化选举器并启动选主
func Init(cfg *Config, l *logger.Logger, opts ...Option) error {
	log = l

	if cfg == nil {
		log.Info("elect config is nil, skipping election init")
		return nil
	}

	var err error
	elector, err = New(cfg, opts...)
	if err != nil {
		return err
	}

	return nil
}

// IsLeader 当前实例是否为 Leader
func IsLeader() bool {
	if elector == nil {
		return false
	}
	return elector.IsLeader()
}

// LeaderID 当前 Leader 的标识
func LeaderID() string {
	if elector == nil {
		return ""
	}
	return elector.LeaderID()
}

// Close 优雅退出
func Close() {
	if elector != nil {
		elector.Close()
	}
}
