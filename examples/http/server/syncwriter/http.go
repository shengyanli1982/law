package main

import (
	"net/http"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// 创建一个 zapcore.EncoderConfig 结构体，用于配置日志的编码器
	// Create a zapcore.EncoderConfig struct to configure the log encoder
	encoderCfg := zapcore.EncoderConfig{
		// MessageKey 设置日志消息的键名
		// MessageKey sets the key name of the log message
		MessageKey: "msg",

		// LevelKey 设置日志级别的键名
		// LevelKey sets the key name of the log level
		LevelKey: "level",

		// NameKey 设置日志记录器名称的键名
		// NameKey sets the key name of the logger name
		NameKey: "logger",

		// EncodeLevel 设置日志级别的编码器，这里使用的是小写编码器
		// EncodeLevel sets the encoder of the log level, here we use the lowercase encoder
		EncodeLevel: zapcore.LowercaseLevelEncoder,

		// EncodeTime 设置时间的编码器，这里使用的是 ISO8601 格式
		// EncodeTime sets the encoder of the time, here we use the ISO8601 format
		EncodeTime: zapcore.ISO8601TimeEncoder,

		// EncodeDuration 设置持续时间的编码器，这里使用的是字符串编码器
		// EncodeDuration sets the encoder of the duration, here we use the string encoder
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	// 创建一个 zapcore.WriteSyncer，用于将日志写入到 os.Stdout
	// Create a zapcore.WriteSyncer to write logs to os.Stdout
	zapSyncWriter := zapcore.AddSync(os.Stdout)

	// 创建一个 zapcore.Core，用于处理日志的核心功能
	// Create a zapcore.Core to handle the core functionality of the logs
	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapSyncWriter, zapcore.DebugLevel)

	// 创建一个 zap.Logger，用于记录日志
	// Create a zap.Logger to log
	zapLogger := zap.New(zapCore)

	// 使用 http.HandleFunc 函数注册一个处理函数，当访问 "/" 路径时，这个函数会被调用
	// Use the http.HandleFunc function to register a handler function, this function will be called when the "/" path is accessed
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 在处理函数中，使用 zapLogger 记录一条信息级别的日志
		// In the handler function, use zapLogger to log an info level log
		zapLogger.Info("hello")
	})

	// 使用 http.ListenAndServe 函数启动一个 HTTP 服务器，监听 8080 端口
	// Use the http.ListenAndServe function to start an HTTP server, listening on port 8080
	_ = http.ListenAndServe(":8080", nil)
}
