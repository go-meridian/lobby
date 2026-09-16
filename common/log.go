package common

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"Lobby/config"
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

	output := &lumberjack.Logger{
		Filename:   fmt.Sprintf("logs/%s.log", cfg.Log.LogFile),
		MaxSize:    500,
		MaxBackups: 30,
		MaxAge:     1,
		Compress:   true,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig.EncoderConfig),
		zapcore.AddSync(output),
		encoderConfig.Level,
	)

	return zap.New(core), nil
}
