package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/edygunawan/go-rest-api/internal/config"
	"github.com/edygunawan/go-rest-api/internal/database"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(args []string) error {
	steps := 0
	if len(args) == 1 && args[0] == "up" {
	} else if len(args) == 2 && args[0] == "down" {
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 1 {
			return fmt.Errorf("down requires a positive step count")
		}
		steps = -n
	} else {
		return fmt.Errorf("usage: migrate up | migrate down <steps>")
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	if err := database.Migrate(cfg.Database, steps); err != nil {
		return err
	}
	fmt.Println("Migrations completed")
	return nil
}
