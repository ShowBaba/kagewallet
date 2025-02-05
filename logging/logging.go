package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitializeLogger(level zapcore.Level) {
	switch level {
	case zapcore.InfoLevel:
		Logger, _ = zap.NewProduction()
	case zapcore.DebugLevel:
		Logger, _ = zap.NewDevelopment()
	}
}

func Debug(msg string, fields ...zapcore.Field) {
	if Logger != nil {
		Logger.Debug(msg, fields...)
	}
}

func Info(msg string, fields ...zapcore.Field) {
	if Logger != nil {
		Logger.Info(msg, fields...)
	}
}

func Warn(msg string, fields ...zapcore.Field) {
	if Logger != nil {
		Logger.Warn(msg, fields...)
	}
}

func Error(msg string, fields ...zapcore.Field) {
	if Logger != nil {
		Logger.Error(msg, fields...)
	}
}

func Fatal(msg string, fields ...zapcore.Field) {
	if Logger != nil {
		Logger.Fatal(msg, fields...)
	}
}
