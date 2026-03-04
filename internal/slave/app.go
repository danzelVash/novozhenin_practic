package slave

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novozhenin/practic/internal/slave/client"
	"github.com/novozhenin/practic/internal/slave/servo"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	// Цикл подключения к master с переподключением
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := a.connectAndListen(ctx); err != nil {
			log.Printf("[slave] соединение потеряно: %v", err)
			log.Println("[slave] переподключение через 3 секунды...")
			time.Sleep(3 * time.Second)
		}
	}
}

// connectAndListen подключается к master и слушает команды.
func (a *App) connectAndListen(ctx context.Context) error {
	conn, err := grpc.NewClient(a.cfg.MasterAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("подключение к master: %w", err)
	}
	defer conn.Close()

	c := client.New(conn, func(directionUp bool) {
		if directionUp {
			if err := a.servo.MoveUp(); err != nil {
				log.Printf("[slave] ошибка движения вверх: %v", err)
			}
		} else {
			if err := a.servo.MoveDown(); err != nil {
				log.Printf("[slave] ошибка движения вниз: %v", err)
			}
		}
	})

	return c.Run(ctx)
}
