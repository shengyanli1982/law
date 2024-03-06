package main

import (
	"net/http"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	x "github.com/shengyanli1982/law"
)

func main() {
	// 创建一个新的异步写入器，输出到标准输出
	// Create a new asynchronous writer that outputs to standard output
	aw := x.NewWriteAsyncer(os.Stdout, nil)
	// 确保在程序结束时停止异步写入器
	// Ensure the asynchronous writer is stopped when the program ends
	defer aw.Stop()

	// 创建一个zapcore.EncoderConfig，用于配置日志编码器
	// Create a zapcore.EncoderConfig to configure the log encoder
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",                         // 消息的键名，Key name for the message
		LevelKey:       "level",                       // 日志级别的键名，Key name for the log level
		NameKey:        "logger",                      // 记录器名称的键名，Key name for the logger name
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 日志级别的编码器，Encoder for the log level
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // 时间的编码器，Encoder for the time
		EncodeDuration: zapcore.StringDurationEncoder, // 持续时间的编码器，Encoder for the duration
	}

	// 创建一个zapcore.WriteSyncer，将日志写入异步写入器
	// Create a zapcore.WriteSyncer that writes logs to the asynchronous writer
	zapSyncWriter := zapcore.AddSync(aw)
	// 创建一个zapcore.Core，使用JSON编码器和异步写入器
	// Create a zapcore.Core using the JSON encoder and the asynchronous writer
	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapSyncWriter, zapcore.DebugLevel)
	// 创建一个zap.Logger，使用上面创建的zapcore.Core
	// Create a zap.Logger using the zapcore.Core created above
	zapLogger := zap.New(zapCore)

	// 注册一个HTTP处理函数，当访问"/"时，记录一条信息日志
	// Register an HTTP handler function, when accessing "/", log an info message
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		zapLogger.Info("hello")
	})
	// 启动HTTP服务器，监听8080端口
	// Start the HTTP server, listen on port 8080
	_ = http.ListenAndServe(":8080", nil)
}
