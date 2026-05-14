package logx

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Fields map[string]any

var logger *zap.Logger

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.TimeKey = "ts"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	built, err := config.Build()
	if err != nil {
		logger = zap.NewNop()
		return
	}

	logger = built
}

func Info(message string, fields Fields) {
	logger.Info(message, toZapFields(fields)...)
}

func Error(message string, fields Fields) {
	logger.Error(message, toZapFields(fields)...)
}

func Fatal(message string, fields Fields) {
	logger.Fatal(message, toZapFields(fields)...)
}

func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}

func toZapFields(fields Fields) []zap.Field {
	if len(fields) == 0 {
		return nil
	}

	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}
	return zapFields
}
