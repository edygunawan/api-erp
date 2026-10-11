package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/edygunawan/go-rest-api/internal/app"
	"github.com/edygunawan/go-rest-api/internal/config"
	"github.com/edygunawan/go-rest-api/internal/usecase/bootstrap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	req := bootstrap.Request{FullName: os.Getenv("ADMIN_FULL_NAME"), Username: os.Getenv("ADMIN_USERNAME"), Password: os.Getenv("ADMIN_PASSWORD")}
	if email := os.Getenv("ADMIN_EMAIL"); email != "" {
		req.Email = &email
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	id, err := app.BootstrapAdmin(ctx, cfg, req)
	if err != nil {
		return err
	}
	fmt.Printf("Admin ready: %s (existing credentials preserved)\n", id)
	return nil
}
