package law

// 本文件是跨轮次性能对标的仓库内 SSOT（Single Source of Truth）。
//
// 运行口径（根包、无 pprof 采样 flag）:
//
//	go test -run=NONE -bench=. -count=5
//
// 跨轮次对比必须同口径：带/不带 pprof 采样（-cpuprofile/-memprofile/-mutexprofile）
// 的 ns/op 相差 15-35%（r2/r3 实测），禁止混用口径直接对比数值。
//
// 覆盖定位（对照 r3 基线报告 .agent-work-b652f803/reports/2026-09-13-law-perf-r3-baseline.md
// 的 8 主路径盘点）:
//   - writer_test.go 的 BenchmarkWriteAsyncer 已覆盖: 顺序小写入 / 大写入(large writes) /
//     并发小写入 / string 写入 / 并发混合写入 —— 本文件不重复;
//   - benchmark/（独立模块）为 law-vs-zap 对比套件，不承担内部路径回归防护;
//   - 本文件补齐 r3 认定无回归防护的三条路径: bounded 队列背压、idle 驱动 flush、Stop 关闭。
//
// 约束: 仅使用公共 API（NewWriteAsyncer/NewConfig/NewBoundedQueue/Write/WriteString/Stop），
// 不依赖内部实现细节；并发/延迟类基准先 settle（等消费者落位）再计时。

import (
	"io"
	"testing"
	"time"
)

// regressionSettle 基准计时前等待 poller 消费者落位的时长，
// 避免把启动期瞬态计入稳态数值（与 writer_test.go 既有基准的 settle 实践一致）。
const regressionSettle = 100 * time.Millisecond

// flushSignalSink 每次 Write 向 ch 非阻塞发送一个信号，
// 供 idle flush 基准探测"数据已落盘到底层 sink"事件。
type flushSignalSink struct {
	ch chan struct{}
}

func (s *flushSignalSink) Write(p []byte) (int, error) {
	select {
	case s.ch <- struct{}{}:
	default:
	}
	return len(p), nil
}

// BenchmarkRegression 补齐 r3 前无回归防护的三条主路径基准，
// 每条缺口路径一个子基准，不构造复杂场景。
func BenchmarkRegression(b *testing.B) {
	b.Run("bounded queue backpressure", func(b *testing.B) {
		// 有界队列 + 零成本 sink：并发写入下覆盖 bounded 路径——
		// Write 侧 Available() 探测、Push 侧 isFull/计数簿记与 notFull 信号；
		// 队列打满时即度量背压阻塞等待与唤醒的真实成本，同属防护范围。
		conf := NewConfig().WithQueue(NewBoundedQueue(1024, 0))
		w := NewWriteAsyncer(io.Discard, conf)
		defer w.Stop()

		// settle：等 poller 消费者落位再计时
		time.Sleep(regressionSettle)

		data := []byte("small")
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = w.Write(data)
			}
		})
	})

	b.Run("idle flush", func(b *testing.B) {
		// 小载荷进入 bufio 缓冲（不触发满缓冲预 flush），随后无新写入，
		// idleTimeout 到期后由心跳分支的双时钟判定触发 idle flush 落盘。
		// 每 op = Write + 等待 flush 抵达底层 sink，守护 idle 驱动 flush 路径。
		sink := &flushSignalSink{ch: make(chan struct{}, 1)}
		conf := NewConfig().
			WithIdleTimeout(2 * time.Millisecond).
			WithHeartbeatInterval(time.Millisecond)
		w := NewWriteAsyncer(sink, conf)
		defer w.Stop()

		// settle：等 poller 进入稳态 select 再计时
		time.Sleep(regressionSettle)

		data := []byte("drip")
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if _, err := w.Write(data); err != nil {
				b.Fatal(err)
			}
			<-sink.ch // 等待 idle flush 完成落盘
		}
	})

	b.Run("stop", func(b *testing.B) {
		// 每轮新建实例并预写未落盘数据（构建与写入在计时外），仅计时 Stop：
		// 覆盖 cancel、wg.Wait、CleanQueue 清残、最终 Flush 的完整关闭路径。
		data := []byte("small")
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			w := NewWriteAsyncer(io.Discard, nil)
			for j := 0; j < 8; j++ {
				if _, err := w.Write(data); err != nil {
					b.Fatal(err)
				}
			}
			b.StartTimer()
			w.Stop()
		}
	})
}
