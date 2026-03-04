package main

import (
	"context"
	"log"

	"github.com/novozhenin/practic/internal/slave"
)

func main() {
	cfg := slave.LoadConfig()

	log.Printf("[slave] master=%s", cfg.MasterAddr)

	app := slave.New(cfg)
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("[slave] %v", err)
	}
}
