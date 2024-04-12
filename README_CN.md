[English](./README.md) | 中文

<div align="center">
	<h1>LAW</h1>
	<p>一个轻量级的异步IO库，提升日志记录、流式传输等操作的性能和效率。</p>
	<img src="assets/logo.png" alt="logo" width="450px">
</div>

[![Go Report Card](https://goreportcard.com/badge/github.com/shengyanli1982/law)](https://goreportcard.com/report/github.com/shengyanli1982/law)
[![Build Status](https://github.com/shengyanli1982/law/actions/workflows/test.yaml/badge.svg)](https://github.com/shengyanli1982/law/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/shengyanli1982/law.svg)](https://pkg.go.dev/github.com/shengyanli1982/law)

# 简介

**Log Asynchronous Writer** 是一个专为高并发场景设计的轻量级日志异步写入器，例如 HTTP 服务器和 gRPC 服务器。

`LAW` 采用了双缓冲设计，使其能够异步地将数据写入双端队列，并在缓冲区满时将缓冲区刷新到 `io.Writer` 中。这种设计显著提高了写入器的性能，并减轻了对 `io.Writer` 的压力。

`LAW` 只有两个 API，`Write` 和 `Stop`，提供了简单易用的操作方式。`Write` API 用于将日志数据写入缓冲区，而 `Stop` API 用于停止写入器。

`LAW` 可以与任何需要异步写入的 `io.Writer` 接口的实现一起使用，例如 `zap`、`logrus`、`klog`、`zerolog` 等。

# 优势

-   简单易用
-   无第三方依赖
-   高性能且低内存占用
-   优化了垃圾回收
-   支持操作回调函数

# 安装

```bash
go get github.com/shengyanli1982/law
```

# 快速入门

`LAW` 的设计简单易用。要开始使用，创建一个写入器并使用 `Write` 方法将日志数据写入缓冲区。当你准备停止写入器时，只需调用 `Stop` 方法即可。

`LAW` 还提供了一个 `Config` 结构体，允许你自定义写入器的行为。你可以使用 `WithXXX` 方法来配置各种功能。更多详情，请参考 **特性** 部分。

### 示例

```go
package main

import (
	"os"
	"time"
	"strconv"

	law "github.com/shengyanli1982/law"
)

func main() {
	// 创建一个新的配置
	// Create a new configuration
	conf := NewConfig()

	// 使用 os.Stdout 和配置创建一个新的 WriteAsyncer 实例
	// Create a new WriteAsyncer instance using os.Stdout and the configuration
	w := NewWriteAsyncer(os.Stdout, conf)
	// 使用 defer 语句确保在 main 函数退出时停止 WriteAsyncer
	// Use a defer statement to ensure that WriteAsyncer is stopped when the main function exits
	defer w.Stop()

	// 循环 10 次，每次都将一个数字写入 WriteAsyncer
	// Loop 10 times, each time write a number to WriteAsyncer
	for i := 0; i < 10; i++ {
		_, _ = w.Write([]byte(strconv.Itoa(i))) // 将当前的数字写入 WriteAsyncer
	}

	// 等待 1 秒，以便我们可以看到 WriteAsyncer 的输出
	// Wait for 1 second so we can see the output of WriteAsyncer
	time.Sleep(time.Second)
}
```

# 特性

`LAW` 还具有一些有趣的特性。它被设计为易于扩展，这意味着您可以轻松编写自己的异步写入器。

## 1. 回调函数

`LAW` 支持回调函数。在创建写入器时，您可以指定一个回调函数，当写入器执行特定操作时，回调函数将被调用。

> [!TIP]
> 回调函数是可选的。如果您不需要回调函数，可以在创建写入器时传递 `nil`，回调函数将不会被调用。
>
> 您可以使用 `WithCallback` 方法来设置回调函数。

### 示例

```go
package main

import (
	"os"
	"time"
	"strconv"

	law "github.com/shengyanli1982/law"
)

// callback 是一个实现了 law.Callback 接口的结构体
// callback is a struct that implements the law.Callback interface
type callback struct{}

// OnPushQueue 是当数据被推入队列时的回调函数
// OnPushQueue is the callback function when data is pushed into the queue
func (c *callback) OnPushQueue(b []byte) {
	fmt.Printf("push queue msg: %s\n", string(b)) // 输出推入队列的消息
}

// OnPopQueue 是当数据从队列中弹出时的回调函数
// OnPopQueue is the callback function when data is popped from the queue
func (c *callback) OnPopQueue(b []byte, lantcy int64) {
	fmt.Printf("pop queue msg: %s, lantcy: %d\n", string(b), lantcy) // 输出弹出队列的消息和延迟
}

// OnWriteSuccess 是当数据写入成功时的回调函数
// OnWriteSuccess is the callback function when data writing succeeds
func (c *callback) OnWriteSuccess(b []byte) {
	fmt.Printf("write success msg: %s\n", string(b)) // 输出写入成功的消息
}

// OnWriteFailed 是当数据写入失败时的回调函数
// OnWriteFailed is the callback function when data writing fails
func (c *callback) OnWriteFailed(b []byte, err error) {
	fmt.Printf("write failed msg: %s, err: %v\n", string(b), err) // 输出写入失败的消息和错误
}

func main() {
	// 创建一个新的配置，并设置回调函数
	// Create a new configuration and set the callback function
	conf := NewConfig().WithCallback(&callback{})

	// 使用 os.Stdout 和配置创建一个新的 WriteAsyncer 实例
	// Create a new WriteAsyncer instance using os.Stdout and the configuration
	w := NewWriteAsyncer(os.Stdout, conf)
	// 使用 defer 语句确保在 main 函数退出时停止 WriteAsyncer
	// Use a defer statement to ensure that WriteAsyncer is stopped when the main function exits
	defer w.Stop()

	// 循环 10 次，每次都将一个数字写入 WriteAsyncer
	// Loop 10 times, each time write a number to WriteAsyncer
	for i := 0; i < 10; i++ {
		_, _ = w.Write([]byte(strconv.Itoa(i))) // 将当前的数字写入 WriteAsyncer
	}

	// 等待 1 秒，以便我们可以看到 WriteAsyncer 的输出
	// Wait for 1 second so we can see the output of WriteAsyncer
	time.Sleep(time.Second)
}
```

## 2. 容量

`LAW` 使用双缓冲区来写入日志数据，允许您在创建写入器时指定缓冲区的容量。

> [!TIP]
>
> -   `deque` 的默认容量是无限的，意味着它可以容纳无限量的日志数据。
> -   `bufferIo` 的默认容量是 `2k`，意味着它可以容纳最多 `2k` 的日志数据。如果缓冲区已满，`LAW` 将自动将缓冲区刷新到 `io.Writer`。`2k` 是一个推荐的选择，但您可以自定义它。
>
> 您可以使用 `WithBufferSize` 方法来更改缓冲区的大小。

### 示例

```go
package main

import (
	"os"
	"time"
	"strconv"

	law "github.com/shengyanli1982/law"
)

func main() {
	// 创建一个新的配置，并设置缓冲区大小为 1024
	// Create a new configuration and set the buffer size to 1024
	conf := NewConfig().WithBufferSize(1024)

	// 使用 os.Stdout 和配置创建一个新的 WriteAsyncer 实例
	// Create a new WriteAsyncer instance using os.Stdout and the configuration
	w := NewWriteAsyncer(os.Stdout, conf)
	// 使用 defer 语句确保在 main 函数退出时停止 WriteAsyncer
	// Use a defer statement to ensure that WriteAsyncer is stopped when the main function exits
	defer w.Stop()

	// 循环 10 次，每次都将一个数字写入 WriteAsyncer
	// Loop 10 times, each time write a number to WriteAsyncer
	for i := 0; i < 10; i++ {
		_, _ = w.Write([]byte(strconv.Itoa(i))) // 将当前的数字写入 WriteAsyncer
	}

	// 等待 1 秒，以便我们可以看到 WriteAsyncer 的输出
	// Wait for 1 second so we can see the output of WriteAsyncer
	time.Sleep(time.Second)
}
```

# 示例

以下是使用 LAW 的一些示例。您还可以参考 `examples` 目录中的更多示例。

## 1. Zap

`LAW` 允许您异步地将日志数据写入 `zap`。

**代码**

```go
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
```

**执行结果**

```bash
$ go run demo.go
{"level":"info","msg":"0"}
{"level":"info","msg":"1"}
{"level":"info","msg":"2"}
{"level":"info","msg":"3"}
{"level":"info","msg":"4"}
{"level":"info","msg":"5"}
{"level":"info","msg":"6"}
{"level":"info","msg":"7"}
{"level":"info","msg":"8"}
{"level":"info","msg":"9"}
```

## 2. Logrus

`LAW`可以异步地将日志数据写入`logrus`。

**代码**

```go
package main

import (
	"os"
	"time"

	law "github.com/shengyanli1982/law"
	"github.com/sirupsen/logrus"
)

func main() {
	// 使用 os.Stdout 创建一个新的 WriteAsyncer 实例
	// Create a new WriteAsyncer instance using os.Stdout
	aw := law.NewWriteAsyncer(os.Stdout, nil)
	// 使用 defer 语句确保在 main 函数退出时停止 WriteAsyncer
	// Use a defer statement to ensure that WriteAsyncer is stopped when the main function exits
	defer aw.Stop()

	// 将 logrus 的输出设置为我们创建的 WriteAsyncer
	// Set the output of logrus to the WriteAsyncer we created
	logrus.SetOutput(aw)

	// 循环 10 次，每次都使用 logrus 输出一个数字
	// Loop 10 times, each time output a number using logrus
	for i := 0; i < 10; i++ {
		logrus.Info(i) // 输出当前的数字
	}

	// 等待 3 秒，以便我们可以看到 logrus 的输出
	// Wait for 3 seconds so we can see the output of logrus
	time.Sleep(3 * time.Second)
}
```

**执行结果**

```bash
$ go run demo.go
time="2023-12-16T12:38:13+08:00" level=info msg=0
time="2023-12-16T12:38:13+08:00" level=info msg=1
time="2023-12-16T12:38:13+08:00" level=info msg=2
time="2023-12-16T12:38:13+08:00" level=info msg=3
time="2023-12-16T12:38:13+08:00" level=info msg=4
time="2023-12-16T12:38:13+08:00" level=info msg=5
time="2023-12-16T12:38:13+08:00" level=info msg=6
time="2023-12-16T12:38:13+08:00" level=info msg=7
time="2023-12-16T12:38:13+08:00" level=info msg=8
time="2023-12-16T12:38:13+08:00" level=info msg=9
```

## 3. klog

`LAW`可以异步地将日志数据写入`klog`。

**代码**

```go
package main

import (
	"os"
	"time"

	law "github.com/shengyanli1982/law"
	"k8s.io/klog/v2"
)

func main() {
	// 使用 os.Stdout 创建一个新的 WriteAsyncer 实例
	// Create a new WriteAsyncer instance using os.Stdout
	aw := law.NewWriteAsyncer(os.Stdout, nil)
	// 使用 defer 语句确保在 main 函数退出时停止 WriteAsyncer
	// Use a defer statement to ensure that WriteAsyncer is stopped when the main function exits
	defer aw.Stop()

	// 将 klog 的输出设置为我们创建的 WriteAsyncer
	// Set the output of klog to the WriteAsyncer we created
	klog.SetOutput(aw)

	// 循环 10 次，每次都使用 klog 输出一个数字
	// Loop 10 times, each time output a number using klog
	for i := 0; i < 10; i++ {
		klog.Info(i) // 输出当前的数字
	}

	// 等待 3 秒，以便我们可以看到 klog 的输出
	// Wait for 3 seconds so we can see the output of klog
	time.Sleep(3 * time.Second)
}
```

**执行结果**

```bash
$ go run demo.go
I1216 12:36:07.637943   17388 demo.go:18] 0
I1216 12:36:07.638105   17388 demo.go:18] 1
I1216 12:36:07.638109   17388 demo.go:18] 2
I1216 12:36:07.638113   17388 demo.go:18] 3
I1216 12:36:07.638117   17388 demo.go:18] 4
I1216 12:36:07.638121   17388 demo.go:18] 5
I1216 12:36:07.638125   17388 demo.go:18] 6
I1216 12:36:07.638128   17388 demo.go:18] 7
I1216 12:36:07.638132   17388 demo.go:18] 8
I1216 12:36:07.638136   17388 demo.go:18] 9
```

## 4. Zerolog

`LAW`可以异步地将日志数据写入`zerolog`。

**代码**

```go
package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	law "github.com/shengyanli1982/law"
)

func main() {
	// 使用 os.Stdout 创建一个新的 WriteAsyncer 实例
	// Create a new WriteAsyncer instance using os.Stdout
	aw := law.NewWriteAsyncer(os.Stdout, nil)
	// 使用 defer 语句确保在 main 函数退出时停止 WriteAsyncer
	// Use a defer statement to ensure that WriteAsyncer is stopped when the main function exits
	defer aw.Stop()

	// 使用 WriteAsyncer 创建一个新的 zerolog.Logger 实例，并添加时间戳
	// Create a new zerolog.Logger instance using WriteAsyncer and add a timestamp
	log := zerolog.New(aw).With().Timestamp().Logger()

	// 循环 10 次，每次都使用 log 输出一个数字和一条消息
	// Loop 10 times, each time output a number and a message using log
	for i := 0; i < 10; i++ {
		log.Info().Int("i", i).Msg("hello") // 输出当前的数字和一条消息
	}

	// 等待 3 秒，以便我们可以看到 log 的输出
	// Wait for 3 seconds so we can see the output of log
	time.Sleep(3 * time.Second)
}
```

**执行结果**

```bash
$ go run demo.go
{"level":"info","i":0,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
{"level":"info","i":1,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
{"level":"info","i":2,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
{"level":"info","i":3,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
{"level":"info","i":4,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
{"level":"info","i":5,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
{"level":"info","i":6,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
{"level":"info","i":7,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
{"level":"info","i":8,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
{"level":"info","i":9,"time":"2023-12-16T12:39:45+08:00","message":"hello"}
```

# 基准测试

> [!IMPORTANT]
> 基准测试结果仅供参考。请注意，不同的硬件环境可能会产生不同的结果。

### 环境

-   **操作系统**: macOS Big Sur 11.7.10
-   **CPU**: 3.3 GHz 8-Core Intel XEON E5 4627v2
-   **内存**: 32 GB 1866 MHz DDR3
-   **Go 版本**: go1.20.11 darwin/amd64

## 1. 基准测试前

自版本 `v0.1.3` 起，`LAW` 的性能已经优化和改进，相比于 `BlackHoleWriter` 和 `zapcore.AddSync(BlackHoleWriter)`。

**在此之前**

```bash
# go test -benchmem -run=^$ -bench ^Benchmark* github.com/shengyanli1982/law/benchmark

goos: darwin
goarch: amd64
pkg: github.com/shengyanli1982/law/benchmark
cpu: Intel(R) Xeon(R) CPU E5-4627 v2 @ 3.30GHz
BenchmarkBlackHoleWriter-8           	1000000000	         0.2871 ns/op	       0 B/op	       0 allocs/op
BenchmarkBlackHoleWriterParallel-8   	1000000000	         0.2489 ns/op	       0 B/op	       0 allocs/op
BenchmarkZapSyncWriter-8             	 3357697	       351.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkZapSyncWriterParallel-8     	21949550	        59.52 ns/op	       0 B/op	       0 allocs/op
BenchmarkZapAsyncWriter-8            	  481237	      2133 ns/op	     932 B/op	       1 allocs/op
BenchmarkZapAsyncWriterParallel-8    	 1453645	       865.7 ns/op	    2074 B/op	       3 allocs/op
```

**在此之后**

```bash
# go test -benchmem -run=^$ -bench ^Benchmark* github.com/shengyanli1982/law/benchmark

goos: darwin
goarch: amd64
pkg: github.com/shengyanli1982/law/benchmark
cpu: Intel(R) Xeon(R) CPU E5-4627 v2 @ 3.30GHz
BenchmarkBlackHoleWriter-8           	1000000000	         0.2905 ns/op	       0 B/op	       0 allocs/op
BenchmarkBlackHoleWriterParallel-8   	1000000000	         0.2557 ns/op	       0 B/op	       0 allocs/op
BenchmarkLogAsyncWriter-8            	 4515822	       229.1 ns/op	      61 B/op	       3 allocs/op
BenchmarkLogAsyncWriterParallel-8    	 4604298	       251.1 ns/op	      61 B/op	       3 allocs/op
BenchmarkZapSyncWriter-8             	 3294104	       352.9 ns/op	       0 B/op	       0 allocs/op
BenchmarkZapSyncWriterParallel-8     	23504499	        59.52 ns/op	       0 B/op	       0 allocs/op
BenchmarkZapAsyncWriter-8            	 2173760	       551.0 ns/op	      56 B/op	       2 allocs/op
BenchmarkZapAsyncWriterParallel-8    	 4663755	       258.1 ns/op	      56 B/op	       2 allocs/op
```

`LAW`采用了双缓冲策略进行日志记录，与 `zapcore.AddSync(BlackHoleWriter)` 相比，这可能会稍微影响性能。这是因为当 `LAW` 与 `zap` 集成时，它间接地利用了 zap 的写入缓冲区。`zap` 通过一个 `deque` 将数据传递给 `LAW`，然后将其刷新到 `io.Writer`（`BlackHoleWriter`）。因此，`LAW`的性能是 `BenchmarkZapSyncWriter` 和 `BenchmarkLogAsyncWriter` 的总和，相当于 `BenchmarkZapAsyncWriter`。

## 2. HTTP 服务器

将`law`集成到 HTTP 服务器中，以模拟真实的业务场景，并将其性能与其他日志记录器进行比较。

### 2.1. SyncWriter

**SyncWriter**: `os.Stdout`

```go
package main

import (
	"net/http"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
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

	// 创建一个zapcore.WriteSyncer，将日志写入标准输出
	// Create a zapcore.WriteSyncer that writes logs to the standard output
	zapSyncWriter := zapcore.AddSync(os.Stdout)
	// 创建一个zapcore.Core，使用JSON编码器和标准输出
	// Create a zapcore.Core using the JSON encoder and the standard output
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
```

使用 `wrk` 工具测试 HTTP 服务器的性能。

```bash
#!/bin/bash

times=0

while [ $times -lt 5 ]
do
    wrk -c 500 -t 10 http://127.0.0.1:8080
    times=$[$times+1]
    sleep 2
    echo "--------------------------------------"
done
```

**执行结果**

![sync](examples/http/server/pics/syncwriter.png)

### 2.2. AsyncWriter

**LAW**: `NewWriteAsyncer(os.Stdout, nil)`

```go
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
```

使用 `wrk` 工具测试 HTTP 服务器的性能。

```bash
#!/bin/bash

times=0

while [ $times -lt 5 ]
do
    wrk -c 500 -t 10 http://127.0.0.1:8080
    times=$[$times+1]
    sleep 2
    echo "--------------------------------------"
done
```

**执行结果**

![sync](examples/http/server/pics/asyncwriter.png)
