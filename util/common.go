package util

import (
	"os"
	"reflect"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Logger       *zap.Logger
	logAtomLevel zap.AtomicLevel
	logPaths     []string
)

func InitLogger(newLogPaths []string) {
	if reflect.DeepEqual(logPaths, newLogPaths) {
		return
	}
	logAtomLevel = zap.NewAtomicLevel()
	logPaths = newLogPaths
	var syncers []zapcore.WriteSyncer
	for _, p := range logPaths {
		switch p {
		case "stdout":
			syncers = append(syncers, zapcore.AddSync(os.Stdout))
		case "stderr":
			syncers = append(syncers, zapcore.AddSync(os.Stderr))
		default:
			writeFile := zapcore.AddSync(&lumberjack.Logger{
				Filename:   p,
				MaxSize:    100, // megabytes
				MaxBackups: 10,
				LocalTime:  true,
			})
			syncers = append(syncers, writeFile)
		}
	}

	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.NewMultiWriteSyncer(syncers...),
		logAtomLevel,
	)
	Logger = zap.New(core, zap.AddStacktrace(zap.ErrorLevel))
}

func SetLogLevel(newLogLevel string) {
	if Logger != nil {
		var lvl zapcore.Level
		if err := lvl.Set(newLogLevel); err != nil {
			lvl = zap.InfoLevel
		}
		logAtomLevel.SetLevel(lvl)
	}
}
