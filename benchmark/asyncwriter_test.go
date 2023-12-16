package benchmark

import (
	"testing"

	x "github.com/shengyanli1982/law"
	xu "github.com/shengyanli1982/law/internal/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var encoderCfg = zapcore.EncoderConfig{
	MessageKey:     "msg",
	LevelKey:       "level",
	NameKey:        "logger",
	EncodeLevel:    zapcore.LowercaseLevelEncoder,
	EncodeTime:     zapcore.ISO8601TimeEncoder,
	EncodeDuration: zapcore.StringDurationEncoder,
}

func BenchmarkBlackHoleWriter(b *testing.B) {
	w := xu.BlackHoleWriter{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = w.Write([]byte("hello"))
	}
}

func BenchmarkBlackHoleWriterParallel(b *testing.B) {
	w := xu.BlackHoleWriter{}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = w.Write([]byte("hello"))
		}
	})
}

func BenchmarkZapSyncWriter(b *testing.B) {
	w := xu.BlackHoleWriter{}

	zapSyncWriter := zapcore.AddSync(&w)
	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapSyncWriter, zapcore.DebugLevel)
	zapLogger := zap.New(zapCore)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		zapLogger.Info("hello")
	}
}
func BenchmarkZapSyncWriterParallel(b *testing.B) {
	w := xu.BlackHoleWriter{}

	zapSyncWriter := zapcore.AddSync(&w)
	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapSyncWriter, zapcore.DebugLevel)
	zapLogger := zap.New(zapCore)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			zapLogger.Info("hello")
		}
	})
}

func BenchmarkZapAsyncWriter(b *testing.B) {
	w := xu.BlackHoleWriter{}

	aw := x.NewWriteAsyncer(&w, nil)
	defer aw.Stop()

	zapAsyncWriter := zapcore.AddSync(aw)
	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapAsyncWriter, zapcore.DebugLevel)
	zapLogger := zap.New(zapCore)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		zapLogger.Info("hello")
	}
}

func BenchmarkZapAsyncWriterParallel(b *testing.B) {
	w := xu.BlackHoleWriter{}

	aw := x.NewWriteAsyncer(&w, nil)
	defer aw.Stop()

	zapAsyncWriter := zapcore.AddSync(aw)
	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapAsyncWriter, zapcore.DebugLevel)
	zapLogger := zap.New(zapCore)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			zapLogger.Info("hello")
		}
	})
}
