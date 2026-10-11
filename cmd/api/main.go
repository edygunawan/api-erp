package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/edygunawan/go-rest-api/internal/app"
	"github.com/edygunawan/go-rest-api/internal/config"
)

func main() {
	// ------------------------------------------------
	// Config
	// ------------------------------------------------
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	// ------------------------------------------------
	// App
	// ------------------------------------------------
	app, err := app.New(cfg)
	if err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := app.Run(ctx); err != nil {
		panic(err)
	}
}
