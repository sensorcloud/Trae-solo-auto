package logging

import (
	"os"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() *zap.Logger {
	logLevel := viper.GetString("log.level")
	logFormat := viper.GetString("log.format")

	var config zap.Config

	if logFormat == "json" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	config.Level = level

	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return logger
}

func GetLogger(name string) *zap.Logger {
	return NewLogger().Named(name)
}

func Sync(logger *zap.Logger) {
	if err := logger.Sync(); err != nil {
		os.Stderr.WriteString("failed to sync logger: " + err.Error() + "\n")
	}
}