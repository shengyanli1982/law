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

	log := zerolog.New(os.Stderr).With().Timestamp().Logger()

	for i := 0; i < 10; i++ {
		log.Info().Int("i", i).Msg("hello")
	}

	time.Sleep(3 * time.Second)
}
