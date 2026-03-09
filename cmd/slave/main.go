package main

import (
	"context"
	"log"

	"github.com/novozhenin/practic/internal/slave"
)

func main() {
	cfg := slave.LoadConfig()

	log.Printf("[slave] connect=%s transport=%s master=%s cable_listen=%s",
		cfg.Connect, cfg.Transport, cfg.MasterAddr, cfg.CableListen)

	app := slave.New(cfg)
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("[slave] %v", err)
	}
}
