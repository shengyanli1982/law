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
