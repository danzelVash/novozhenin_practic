package slave

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novozhenin/practic/internal/slave/servo"
	"github.com/novozhenin/practic/internal/transport"
	"github.com/novozhenin/practic/internal/transport/grpctransport"
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

	// Инициализация сервопривода
	a.servo = servo.New()
	if err := a.servo.Init(); err != nil {
		return fmt.Errorf("инициализация серво: %w", err)
	}
	defer a.servo.Close()

	// Транспорт (gRPC; в будущем — websocket, MQTT)
	sub := grpctransport.NewSubscriber(a.cfg.MasterAddr)

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
	if cmd.DirectionUp {
		if err := a.servo.MoveUp(); err != nil {
			log.Printf("[slave] ошибка движения вверх: %v", err)
		}
	} else {
		if err := a.servo.MoveDown(); err != nil {
			log.Printf("[slave] ошибка движения вниз: %v", err)
		}
	}
}
