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
