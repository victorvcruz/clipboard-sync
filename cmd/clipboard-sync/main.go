package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/victorvcruz/clipboard-sync/internal/app"
)

var version = "dev"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application := app.New(version)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	if err := application.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
