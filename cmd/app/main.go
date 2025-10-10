package main

import (
	"log"

	"github.com/bruhabruh/file-hosting/internal/app"
	"github.com/bruhabruh/file-hosting/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app := app.New(cfg)
	app.Run()
}
