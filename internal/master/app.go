package master

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novozhenin/practic/internal/master/neuro"
	"github.com/novozhenin/practic/internal/master/recorder"
	"github.com/novozhenin/practic/internal/master/server"
	"github.com/novozhenin/practic/internal/master/vad"
	pb "github.com/novozhenin/practic/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// App — главное приложение master-сервиса.
type App struct {
	cfg      Config
	recorder *recorder.Recorder
	vad      *vad.VAD
	neuro    *neuro.Gateway
	server   *server.Server
}

// New создаёт приложение master.
func New(cfg Config) *App {
	return &App{cfg: cfg}
}

// Run запускает master-сервис.
func (a *App) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Обработка сигналов
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("[master] завершение...")
		cancel()
	}()

	// Инициализация компонентов
	a.recorder = recorder.New(a.cfg.AudioDevice, a.cfg.AudioRate)
	a.vad = vad.New(a.cfg.VADThreshold,
		time.Duration(a.cfg.SilenceDur*float64(time.Second)),
		a.cfg.AudioRate)

	// Подключение к нейросервису
	neuroConn, err := grpc.NewClient(a.cfg.NeuroAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("подключение к нейросервису: %w", err)
	}
	defer neuroConn.Close()
	a.neuro = neuro.NewGateway(neuroConn)

	// Запуск gRPC-сервера для slave
	a.server = server.New()
	grpcServer := grpc.NewServer()
	pb.RegisterServoControlServer(grpcServer, a.server)

	lis, err := net.Listen("tcp", a.cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	go func() {
		log.Printf("[master] gRPC-сервер запущен на %s", a.cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("[master] gRPC-сервер: %v", err)
		}
	}()
	defer grpcServer.GracefulStop()

	// Запуск конвейера: запись → VAD → нейро → slave
	return a.runPipeline(ctx)
}

// runPipeline запускает конвейер обработки аудио.
func (a *App) runPipeline(ctx context.Context) error {
	reader, err := a.recorder.Start(ctx)
	if err != nil {
		return fmt.Errorf("запуск записи: %w", err)
	}
	defer a.recorder.Stop()

	phrases := make(chan []byte, 8)

	go func() {
		a.vad.Process(reader, phrases)
		close(phrases)
	}()

	for {
		select {
		case phrase, ok := <-phrases:
			if !ok {
				log.Println("[master] VAD канал закрыт")
				return nil
			}

			command, err := a.neuro.Recognize(ctx, phrase)
			if err != nil {
				log.Printf("[master] ошибка распознавания: %v", err)
				continue
			}

			switch command {
			case "вверх":
				a.server.SendCommand(true)
			case "вниз":
				a.server.SendCommand(false)
			default:
				log.Printf("[master] неизвестная команда: %s", command)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
