package master

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novozhenin/practic/internal/cable"
	"github.com/novozhenin/practic/internal/master/neuro"
	"github.com/novozhenin/practic/internal/master/recorder"
	"github.com/novozhenin/practic/internal/master/vad"
	"github.com/novozhenin/practic/internal/transport"
	"github.com/novozhenin/practic/internal/transport/grpctransport"
	"github.com/novozhenin/practic/internal/transport/mqtttransport"
	"github.com/novozhenin/practic/internal/transport/wstransport"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// App — главное приложение master-сервиса.
type App struct {
	cfg       Config
	recorder  *recorder.Recorder
	vad       *vad.VAD
	neuro     *neuro.Gateway
	publisher transport.Publisher
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

	if a.cfg.Connect == "cable" {
		return cable.RunMaster(ctx, cable.MasterConfig{
			SlaveAddr: a.cfg.CableSlave,
		})
	}

	return a.runLegacy(ctx)
}

func (a *App) runLegacy(ctx context.Context) error {
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

	// Запуск транспорта
	var pub transport.Publisher
	var stopTransport func()

	switch a.cfg.Transport {
	case "grpc":
		p, err := grpctransport.NewPublisher(a.cfg.GRPCPort)
		if err != nil {
			return fmt.Errorf("запуск транспорта: %w", err)
		}
		pub, stopTransport = p, p.Stop
	case "websocket":
		p, err := wstransport.NewPublisher(a.cfg.WSPort)
		if err != nil {
			return fmt.Errorf("запуск транспорта: %w", err)
		}
		pub, stopTransport = p, p.Stop
	case "mqtt":
		p, err := mqtttransport.NewPublisher(a.cfg.MQTTBroker)
		if err != nil {
			return fmt.Errorf("запуск транспорта: %w", err)
		}
		pub, stopTransport = p, p.Stop
	default:
		return fmt.Errorf("неизвестный транспорт: %s", a.cfg.Transport)
	}
	defer stopTransport()
	a.publisher = pub

	// Запуск конвейера: запись → VAD → нейро → транспорт
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
				log.Println("[master/pipeline] VAD канал закрыт")
				return nil
			}

			log.Printf("[master/pipeline] получена фраза от VAD: %d байт, отправляю в нейросеть", len(phrase))

			command, err := a.neuro.Recognize(ctx, phrase)
			if err != nil {
				log.Printf("[master/pipeline] ошибка распознавания: %v", err)
				continue
			}

			log.Printf("[master/pipeline] нейросеть вернула команду: %q", command)

			switch command {
			case "вверх":
				log.Println("[master/pipeline] отправка команды slave: вверх")
				a.publisher.Publish(transport.Command{DirectionUp: true})
				log.Println("[master/pipeline] команда отправлена в slave")
			case "вниз":
				log.Println("[master/pipeline] отправка команды slave: вниз")
				a.publisher.Publish(transport.Command{DirectionUp: false})
				log.Println("[master/pipeline] команда отправлена в slave")
			default:
				log.Printf("[master/pipeline] неизвестная команда: %q, пропускаем", command)
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
