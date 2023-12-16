package main

import (
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	law "github.com/shengyanli1982/law"
)

func main() {
	aw := law.NewWriteAsyncer(os.Stdout, nil)
	defer aw.Stop()

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	zapAsyncWriter := zapcore.AddSync(aw)
	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapAsyncWriter, zapcore.DebugLevel)
	zapLogger := zap.New(zapCore)

	for i := 0; i < 10; i++ {
		zapLogger.Info(strconv.Itoa(i))
	}

	time.Sleep(3 * time.Second)
}
