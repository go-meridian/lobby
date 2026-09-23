package logger

import (
	"fmt"
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logValue  atomic.Value // 无锁读取 *zap.Logger，永不返回 nil
	mainW     *LogWriter   // 主日志写入器，用于 Close
	errW      *LogWriter   // 错误日志写入器，用于 Close（可能为 nil）
	closeOnce sync.Once    // 保证 Close 幂等
)

func init() {
	logValue.Store(zap.NewNop())
}

// Init 初始化全局日志，仅可调用一次，重复调用会泄漏前一次的日志文件句柄
func Init(cfg *Config) (*zap.Logger, error) {
	if cfg.LogDir == "" {
		cfg.LogDir = "logs"
	}
	if cfg.LogFile == "" {
		cfg.LogFile = "app"
	}

	encoderConfig := zap.NewProductionConfig()
	encoderConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logLevel, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level error: %w", err)
	}
	encoderConfig.Level = zap.NewAtomicLevelAt(logLevel)

	mainWriter, err := NewLogWriter(cfg.LogDir, cfg.LogFile, cfg.MaxSize, cfg.MaxAge)
	if err != nil {
		return nil, fmt.Errorf("create main log writer error: %w", err)
	}

	mainCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig.EncoderConfig),
		zapcore.AddSync(mainWriter),
		encoderConfig.Level,
	)

	var newLog *zap.Logger
	if cfg.ErrorFile != "" {
		errorWriter, err := NewLogWriter(cfg.LogDir, cfg.ErrorFile, cfg.MaxSize, cfg.MaxAge)
		if err != nil {
			mainWriter.Close()
			return nil, fmt.Errorf("create error log writer error: %w", err)
		}

		errorLevelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		})

		errorCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig.EncoderConfig),
			zapcore.AddSync(errorWriter),
			errorLevelEnabler,
		)

		errW = errorWriter
		newLog = zap.New(zapcore.NewTee(mainCore, errorCore))
	} else {
		newLog = zap.New(mainCore)
	}

	mainW = mainWriter
	logValue.Store(newLog)
	return newLog, nil
}

// InitWithDefault 使用默认配置初始化
func InitWithDefault(logFile string) (*zap.Logger, error) {
	return Init(DefaultConfig(logFile))
}

// Get 获取全局日志实例（无锁原子读取，永不返回 nil）
func Get() *zap.Logger {
	return logValue.Load().(*zap.Logger)
}

// Close 刷新日志缓冲区并关闭所有日志文件（幂等，多次调用安全）
func Close() {
	closeOnce.Do(func() {
		if l, ok := logValue.Load().(*zap.Logger); ok && l != nil {
			l.Sync()
		}
		if mainW != nil {
			mainW.Close()
		}
		if errW != nil {
			errW.Close()
		}
		logValue.Store(zap.NewNop())
	})
}
