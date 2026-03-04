package main

import (
	"context"
	"log"

	"github.com/novozhenin/practic/internal/master"
)

func main() {
	cfg := master.LoadConfig()

	log.Printf("[master] transport=%s neuro=%s grpc=%s device=%s", cfg.Transport, cfg.NeuroAddr, cfg.GRPCPort, cfg.AudioDevice)

	app := master.New(cfg)
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("[master] %v", err)
	}
}
