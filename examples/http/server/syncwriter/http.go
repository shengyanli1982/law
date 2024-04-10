package main

import (
	"net/http"

	"os"

	"go.uber.org/zap"

	"go.uber.org/zap/zapcore"
)

func main() {

	encoderCfg := zapcore.EncoderConfig{

		MessageKey: "msg",

		LevelKey: "level",

		NameKey: "logger",

		EncodeLevel: zapcore.LowercaseLevelEncoder,

		EncodeTime: zapcore.ISO8601TimeEncoder,

		EncodeDuration: zapcore.StringDurationEncoder,
	}

	zapSyncWriter := zapcore.AddSync(os.Stdout)

	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapSyncWriter, zapcore.DebugLevel)

	zapLogger := zap.New(zapCore)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		zapLogger.Info("hello")

	})

	_ = http.ListenAndServe(":8080", nil)

}
