package logger

import (
	"Lobby/config"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLogger(cfg *config.Config, logger echo.Logger) (*zap.Logger, error) {
	encoderConfig := zap.NewProductionConfig()
	encoderConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logLevel, err := zapcore.ParseLevel(cfg.Log.Level)
	if err != nil {
		logger.Error("common.InitLogger zapcore.ParseLevel error! err[ %s ]", err.Error())
		return nil, err
	}
	encoderConfig.Level = zap.NewAtomicLevelAt(logLevel)

	output, err := NewLogWriter("logs", cfg.Log.LogFile, cfg.Log.MaxSize, cfg.Log.MaxAge)
	if err != nil {
		logger.Error("common.InitLogger NewLogWriter error! err[ %s ]", err.Error())
		return nil, err
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig.EncoderConfig),
		zapcore.AddSync(output),
		encoderConfig.Level,
	)

	return zap.New(core), nil
}
