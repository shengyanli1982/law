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
