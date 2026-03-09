package slave

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novozhenin/practic/internal/cable"
	"github.com/novozhenin/practic/internal/slave/servo"
	"github.com/novozhenin/practic/internal/transport"
	"github.com/novozhenin/practic/internal/transport/grpctransport"
	"github.com/novozhenin/practic/internal/transport/mqtttransport"
	"github.com/novozhenin/practic/internal/transport/wstransport"
)

// App — главное приложение slave-сервиса.
type App struct {
	cfg   Config
	servo *servo.Servo
}

// New создаёт приложение slave.
func New(cfg Config) *App {
	return &App{cfg: cfg}
}

// Run запускает slave-сервис.
func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Обработка сигналов
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("[slave] завершение...")
		cancel()
	}()

	if a.cfg.Connect == "cable" {
		a.servo = servo.New()
		if err := a.servo.Init(); err != nil {
			return fmt.Errorf("инициализация серво: %w", err)
		}
		defer a.servo.Close()

		return cable.RunSlave(ctx, cable.SlaveConfig{
			ListenAddr: a.cfg.CableListen,
			OnCommand:  a.handleCableCommand,
		})
	}

	return a.runLegacy(ctx)
}

func (a *App) runLegacy(ctx context.Context) error {
	// Инициализация сервопривода
	a.servo = servo.New()
	if err := a.servo.Init(); err != nil {
		return fmt.Errorf("инициализация серво: %w", err)
	}
	defer a.servo.Close()

	// Транспорт
	var sub transport.Subscriber

	switch a.cfg.Transport {
	case "grpc":
		sub = grpctransport.NewSubscriber(a.cfg.MasterAddr)
	case "websocket":
		sub = wstransport.NewSubscriber(a.cfg.MasterWSURL)
	case "mqtt":
		sub = mqtttransport.NewSubscriber(a.cfg.MQTTBroker)
	default:
		return fmt.Errorf("неизвестный транспорт: %s", a.cfg.Transport)
	}

	// Цикл подключения к master с переподключением
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := sub.Listen(ctx, a.handleCommand); err != nil {
			log.Printf("[slave] соединение потеряно: %v", err)
			log.Println("[slave] переподключение через 3 секунды...")
			time.Sleep(3 * time.Second)
		}
	}
}

// handleCommand обрабатывает полученную команду от master.
func (a *App) handleCommand(cmd transport.Command) {
	direction := "вниз"
	if cmd.DirectionUp {
		direction = "вверх"
	}
	log.Printf("[slave/handler] получена команда: %s, выполняю...", direction)

	if cmd.DirectionUp {
		if err := a.servo.MoveUp(); err != nil {
			log.Printf("[slave/handler] ошибка движения вверх: %v", err)
			return
		}
	} else {
		if err := a.servo.MoveDown(); err != nil {
			log.Printf("[slave/handler] ошибка движения вниз: %v", err)
			return
		}
	}

	log.Printf("[slave/handler] команда %s выполнена", direction)
}

func (a *App) handleCableCommand(command string) error {
	switch command {
	case cable.CommandUp:
		log.Println("[slave/cable-handler] выполняю команду: вверх")
		return a.servo.MoveUp()
	case cable.CommandDown:
		log.Println("[slave/cable-handler] выполняю команду: вниз")
		return a.servo.MoveDown()
	default:
		return fmt.Errorf("неизвестная cable-команда: %s", command)
	}
}
