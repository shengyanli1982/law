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
