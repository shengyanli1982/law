package main

import (
	"net/http"
	"os"

	x "github.com/shengyanli1982/law"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {

	aw := x.NewWriteAsyncer(os.Stdout, nil)

	defer aw.Stop()

	encoderCfg := zapcore.EncoderConfig{

		MessageKey: "msg",

		LevelKey: "level",

		NameKey: "logger",

		EncodeLevel: zapcore.LowercaseLevelEncoder,

		EncodeTime: zapcore.ISO8601TimeEncoder,

		EncodeDuration: zapcore.StringDurationEncoder,
	}

	zapSyncWriter := zapcore.AddSync(aw)

	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapSyncWriter, zapcore.DebugLevel)

	zapLogger := zap.New(zapCore)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		zapLogger.Info("hello")

	})

	_ = http.ListenAndServe(":8080", nil)

}
