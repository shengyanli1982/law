<div align="center">
	<h1>LAW</h1>
	<p>A lightweight elevate I/O with Asynchronous io.Writer<p>
	<p>Boost performance and efficiency effortlessly for logging, streaming, and more.</p>
	<img src="assets/logo.png" alt="logo" width="300px">
    
</div>

# Introduction

**Log Asynchronous Writer** is a lightweight log asynchronous writer. It is designed to be used in high concurrency scenarios, such as HTTP servers, gRPC servers, etc.

`LAW` has `double buffer` design which means that it can write data to the `deque` asynchronously, and then flush the buffer to the `io.Writer` when the buffer writer is full. Split the write action and data writing, which design can greatly improve the performance of the writer and reduce the pressure on the `io.Writer`.

`LAW` is very simple, it only has two APIs: `Write` and `Stop`. `Write` is used to write log data to the buffer, and `Stop` is used to stop the writer.

`LAW` can be used any where which implements the `io.Writer` interface and asynchronous writing is required. such as `zap`, `logrus`, `klog`, `zerolog`, etc.

# Advantage

-   Simple and easy to use
-   No third-party dependencies
-   High performance
-   Low memory usage
-   GC optimization
-   Support action callback functions

# Installation

```bash
go get github.com/shengyanli1982/law
```

# Quick Start

`LAW` is very simple. Just create a writer and use the `Write` method to write log data to the buffer. When you want to stop the writer, just call the `Stop` method.

`LAW` provides a `Config` struct to configure the writer. You can use the `WithXXX` method to configure the writer. Detail see **Features** section.

### Example

```go
w := NewWriteAsyncer(os.Stdout, nil) // create a writer

w.Write([]byte("hello world")) // write log data to the buffer

w.Stop() // stop the writer
```

# Features

`LAW` also has interesting properties. It is designed to be easily extensible which mean you can easily write your own asynchronous writer.

## 1. Callback

`LAW` supports action callback function. Specify a callback functions when create a writer, and the callback function will be called when the writer do some actions.

> [!TIP]
> Callback functions is not required that you can use `LAW` without callback functions. Set `nil` when create a writer, and the callback function will not be called.
>
> You can use `WithCallback` method to set callback functions.

### Example

```go
package main

import (
    "os"
    "time"
    "strconv"

    law "github.com/shengyanli1982/law"
)

type callback struct{}

func (c *callback) OnPushQueue(b []byte) {
    fmt.Printf("push queue msg: %s\n", string(b))
}

func (c *callback) OnPopQueue(b []byte, lantcy int64) {
    fmt.Printf("pop queue msg: %s, lantcy: %d\n", string(b), lantcy)
}

func (c *callback) OnWrite(b []byte) {
    fmt.Printf("write msg: %s\n", string(b))
}

func main() {
    conf := NewConfig().WithCallback(&callback{})

    w := NewWriteAsyncer(os.Stdout, conf)
    defer w.Stop()

    for i := 0; i < 10; i++ {
        _, _ = w.Write([]byte(strconv.Itoa(i)))
    }

    time.Sleep(time.Second)
}
```

## 2. Capacity

`LAW` use `double buffer` to write log data, so you can specify the capacity of the buffer when create a writer.

> [!TIP]
>
> -   The default `deque` capacity is infinity, which means that the buffer can hold unlimited log data.
> -   The default `bufferIo` capacity is `2k`, which means that the buffer can hold `2k` log data. If the capacity of the buffer is full, `LAW` will auto flush the buffer to the `io.Writer`. `2k` is a good choice, but you can also change it.
>
> You can use `WithBufferSize` method to change the bufferIo size.

### Example

```go
package main

import (
    "os"
    "time"
    "strconv"

    law "github.com/shengyanli1982/law"
)

func main() {
    conf := NewConfig().WithBufferSize(1024)

    w := NewWriteAsyncer(os.Stdout, conf)
    defer w.Stop()

    for i := 0; i < 10; i++ {
        _, _ = w.Write([]byte(strconv.Itoa(i)))
    }

    time.Sleep(time.Second)
}
```

# Examples

Here are some examples of how to use `LAW`. but you can also refer to the [examples](examples) directory for more examples.

## 1. Zap

You can use `LAW` to write log data to `zap` asynchronously.

**Code**

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
```

**Results**

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

You can use `LAW` to write log data to `logrus` asynchronously.

**Code**

```go
package main

import (
	"os"
	"time"

	law "github.com/shengyanli1982/law"
	"github.com/sirupsen/logrus"
)

func main() {
	aw := law.NewWriteAsyncer(os.Stdout, nil)
	defer aw.Stop()

	logrus.SetOutput(aw)

	for i := 0; i < 10; i++ {
		logrus.Info(i)
	}

	time.Sleep(3 * time.Second)
}
```

**Results**

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

You can use `LAW` to write log data to `klog` asynchronously.

**Code**

```go
package main

import (
	"os"
	"time"

	law "github.com/shengyanli1982/law"
	"k8s.io/klog/v2"
)

func main() {
	aw := law.NewWriteAsyncer(os.Stdout, nil)
	defer aw.Stop()

	klog.SetOutput(aw)

	for i := 0; i < 10; i++ {
		klog.Info(i)
	}

	time.Sleep(3 * time.Second)
}
```

**Results**

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

You can use `LAW` to write log data to `zerolog` asynchronously.

**Code**

```go
package main

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	law "github.com/shengyanli1982/law"
)

func main() {
	aw := law.NewWriteAsyncer(os.Stdout, nil)
	defer aw.Stop()

	log := zerolog.New(aw).With().Timestamp().Logger()

	for i := 0; i < 10; i++ {
		log.Info().Int("i", i).Msg("hello")
	}

	time.Sleep(3 * time.Second)
}

```

**Results**

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

# Benchmark

> [!IMPORTANT]
> Benchmark test result is only for reference. Different hardware environment will have different results.

### Environment

-   **OS**: macOS Big Sur 11.7.10
-   **CPU**: 3.3 GHz 8-Core Intel XEON E5 4627v2
-   **Memory**: 32 GB 1866 MHz DDR3
-   **Go**: go1.20.11 darwin/amd64

## 1. Base

Compare the performance of `LAW` with `BlackHoleWriter` and `zapcore.AddSync(BlackHoleWriter)`.

Since version `v0.1.3`, the `LAW` codes has been optimized and the performance has been improved.

**Before**

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

**After**

```bash
# go test -benchmem -run=^$ -bench ^Benchmark* github.com/shengyanli1982/law/benchmark

goos: darwin
goarch: amd64
pkg: github.com/shengyanli1982/law/benchmark
cpu: Intel(R) Xeon(R) CPU E5-4627 v2 @ 3.30GHz
BenchmarkBlackHoleWriter-12            	1000000000	         0.2796 ns/op	       0 B/op	       0 allocs/op
BenchmarkBlackHoleWriterParallel-12    	1000000000	         0.2156 ns/op	       0 B/op	       0 allocs/op
BenchmarkLogAsyncWriter-12             	 5184232	       268.9 ns/op	      61 B/op	       3 allocs/op
BenchmarkLogAsyncWriterParallel-12     	 4275908	       244.5 ns/op	      58 B/op	       3 allocs/op
BenchmarkZapSyncWriter-12              	 3242724	       341.0 ns/op	       0 B/op	       0 allocs/op
BenchmarkZapSyncWriterParallel-12      	21295935	        60.57 ns/op	       0 B/op	       0 allocs/op
BenchmarkZapAsyncWriter-12             	 2179002	       694.4 ns/op	      62 B/op	       2 allocs/op
BenchmarkZapAsyncWriterParallel-12     	 5183258	       387.7 ns/op	      73 B/op	       2 allocs/op
```

`LAW` employs a `double buffer` strategy for logging, potentially leading to slightly lower performance compared to `zapcore.AddSync(BlackHoleWriter)`. This is because `zap`, integrating with `LAW`, utilizes zap's writer buffer not directly. `zap` give the data to `LAW`, through a `deque` before flushing to `io.Writer` (`BlackHoleWriter`), resulting in its performance being the sum of `BenchmarkZapSyncWriter` and `BenchmarkLogAsyncWriter`, equivalent to `BenchmarkZapAsyncWriter`.

## 2. Http Server

Integrate law into the http server to simulate real business scenarios and compare the performance of law with other loggers.

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
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
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
```

Use `wrk` to test the performance of the http server.

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

**Results:**

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
	aw := x.NewWriteAsyncer(os.Stdout, nil)
	defer aw.Stop()

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		NameKey:        "logger",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
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
```

Use `wrk` to test the performance of the http server.

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

**Results:**

![sync](examples/http/server/pics/asyncwriter.png)
