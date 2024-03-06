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
	// 使用 os.Stdout 创建一个新的 WriteAsyncer 实例
	// Create a new WriteAsyncer instance using os.Stdout
	aw := law.NewWriteAsyncer(os.Stdout, nil)
	// 使用 defer 语句确保在 main 函数退出时停止 WriteAsyncer
	// Use a defer statement to ensure that WriteAsyncer is stopped when the main function exits
	defer aw.Stop()

	// 创建一个 zapcore.EncoderConfig 实例，用于配置 zap 的编码器
	// Create a zapcore.EncoderConfig instance to configure the encoder of zap
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",                         // 消息的键名
		LevelKey:       "level",                       // 级别的键名
		NameKey:        "logger",                      // 记录器名的键名
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 级别的编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // 时间的编码器
		EncodeDuration: zapcore.StringDurationEncoder, // 持续时间的编码器
	}

	// 使用 WriteAsyncer 创建一个 zapcore.WriteSyncer 实例
	// Create a zapcore.WriteSyncer instance using WriteAsyncer
	zapAsyncWriter := zapcore.AddSync(aw)
	// 使用编码器配置和 WriteSyncer 创建一个 zapcore.Core 实例
	// Create a zapcore.Core instance using the encoder configuration and WriteSyncer
	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapAsyncWriter, zapcore.DebugLevel)
	// 使用 Core 创建一个 zap.Logger 实例
	// Create a zap.Logger instance using Core
	zapLogger := zap.New(zapCore)

	// 循环 10 次，每次都使用 zapLogger 输出一个数字
	// Loop 10 times, each time output a number using zapLogger
	for i := 0; i < 10; i++ {
		zapLogger.Info(strconv.Itoa(i)) // 输出当前的数字
	}

	// 等待 3 秒，以便我们可以看到 zapLogger 的输出
	// Wait for 3 seconds so we can see the output of zapLogger
	time.Sleep(3 * time.Second)
}
