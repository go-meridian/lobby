package logger

import (
	"fmt"

	"lobby/model/codeerror"

	"lobby/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Init 初始化全局日志
func Init(cfg *config.Config) *codeerror.CodeError {
	encoderConfig := zap.NewProductionConfig()
	encoderConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logLevel, err := zapcore.ParseLevel(cfg.Log.Level)
	if err != nil {
		fmt.Printf("logger.Init zapcore.ParseLevel error: %s\n", err.Error())
		return codeerror.LoggerError.Msg("InitLogger zapcore.ParseLevel error: " + err.Error())
	}
	encoderConfig.Level = zap.NewAtomicLevelAt(logLevel)

	output, ce := NewLogWriter("logs", cfg.Log.LogFile, cfg.Log.MaxSize, cfg.Log.MaxAge)
	if ce != nil {
		fmt.Printf("logger.Init NewLogWriter error: %s\n", ce.Error())
		return ce
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig.EncoderConfig),
		zapcore.AddSync(output),
		encoderConfig.Level,
	)

	log = zap.New(core)
	return nil
}

// Get 获取全局日志实例
func Get() *zap.Logger {
	return log
}
