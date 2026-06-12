package benchmark

import (
	"os"
	"testing"
	"time"

	x "github.com/shengyanli1982/law"
	xu "github.com/shengyanli1982/law/internal/utils"
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

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = w.Write([]byte("hello"))
	}
}

func BenchmarkBlackHoleWriterParallel(b *testing.B) {
	w := xu.BlackHoleWriter{}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = w.Write([]byte("hello"))
		}
	})
}
func BenchmarkLogAsyncWriter(b *testing.B) {
	w := xu.BlackHoleWriter{}

	aw := x.NewWriteAsyncer(&w, nil)
	defer aw.Stop()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = aw.Write([]byte("hello"))
	}
}

func BenchmarkLogAsyncWriterParallel(b *testing.B) {
	w := xu.BlackHoleWriter{}

	aw := x.NewWriteAsyncer(&w, nil)
	defer aw.Stop()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = aw.Write([]byte("hello"))
		}
	})
}

func BenchmarkZapSyncWriter(b *testing.B) {
	w := xu.BlackHoleWriter{}

	zapSyncWriter := zapcore.AddSync(&w)
	zapCore := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), zapSyncWriter, zapcore.DebugLevel)
	zapLogger := zap.New(zapCore)

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

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			zapLogger.Info("hello")
		}
	})
}

type slowWriter struct {
	latency time.Duration
}

func (w *slowWriter) Write(p []byte) (int, error) {
	time.Sleep(w.latency)
	return len(p), nil
}

func newZapLogger(ws zapcore.WriteSyncer) *zap.Logger {
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderCfg), ws, zapcore.DebugLevel)
	return zap.New(core)
}

func BenchmarkZapLockedWriter(b *testing.B) {
	w := xu.BlackHoleWriter{}
	ws := zapcore.Lock(zapcore.AddSync(&w))
	logger := newZapLogger(ws)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("hello")
	}
}

func BenchmarkZapLockedWriterParallel(b *testing.B) {
	w := xu.BlackHoleWriter{}
	ws := zapcore.Lock(zapcore.AddSync(&w))
	logger := newZapLogger(ws)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("hello")
		}
	})
}

func BenchmarkSlowIO_ZapLockedParallel(b *testing.B) {
	for _, latency := range []time.Duration{100 * time.Nanosecond, 500 * time.Nanosecond, time.Microsecond} {
		b.Run(latency.String(), func(b *testing.B) {
			w := &slowWriter{latency: latency}
			ws := zapcore.Lock(zapcore.AddSync(w))
			logger := newZapLogger(ws)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					logger.Info("hello")
				}
			})
		})
	}
}

func BenchmarkSlowIO_ZapLockedSerial(b *testing.B) {
	for _, latency := range []time.Duration{100 * time.Nanosecond, 500 * time.Nanosecond, time.Microsecond} {
		b.Run(latency.String(), func(b *testing.B) {
			w := &slowWriter{latency: latency}
			ws := zapcore.Lock(zapcore.AddSync(w))
			logger := newZapLogger(ws)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				logger.Info("hello")
			}
		})
	}
}

func BenchmarkSlowIO_LawAsyncParallel(b *testing.B) {
	for _, latency := range []time.Duration{100 * time.Nanosecond, 500 * time.Nanosecond, time.Microsecond} {
		b.Run(latency.String(), func(b *testing.B) {
			w := &slowWriter{latency: latency}
			aw := x.NewWriteAsyncer(w, nil)
			defer aw.Stop()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = aw.Write([]byte("hello"))
				}
			})
		})
	}
}

func BenchmarkSlowIO_LawAsyncSerial(b *testing.B) {
	for _, latency := range []time.Duration{100 * time.Nanosecond, 500 * time.Nanosecond, time.Microsecond} {
		b.Run(latency.String(), func(b *testing.B) {
			w := &slowWriter{latency: latency}
			aw := x.NewWriteAsyncer(w, nil)
			defer aw.Stop()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = aw.Write([]byte("hello"))
			}
		})
	}
}

func BenchmarkSlowIO_ZapAsyncWithLawParallel(b *testing.B) {
	for _, latency := range []time.Duration{100 * time.Nanosecond, 500 * time.Nanosecond, time.Microsecond} {
		b.Run(latency.String(), func(b *testing.B) {
			w := &slowWriter{latency: latency}
			aw := x.NewWriteAsyncer(w, nil)
			defer aw.Stop()
			ws := zapcore.AddSync(aw)
			logger := newZapLogger(ws)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					logger.Info("hello")
				}
			})
		})
	}
}

func BenchmarkDevNull_ZapLockedParallel(b *testing.B) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Skip(err)
	}
	defer f.Close()
	ws := zapcore.Lock(zapcore.AddSync(f))
	logger := newZapLogger(ws)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("hello")
		}
	})
}

func BenchmarkDevNull_LawAsyncParallel(b *testing.B) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Skip(err)
	}
	defer f.Close()
	aw := x.NewWriteAsyncer(f, nil)
	defer aw.Stop()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = aw.Write([]byte("hello"))
		}
	})
}

func BenchmarkDevNull_ZapAsyncWithLawParallel(b *testing.B) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Skip(err)
	}
	defer f.Close()
	aw := x.NewWriteAsyncer(f, nil)
	defer aw.Stop()
	ws := zapcore.AddSync(aw)
	logger := newZapLogger(ws)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info("hello")
		}
	})
}
