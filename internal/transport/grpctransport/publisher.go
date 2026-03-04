package grpctransport

import (
	"context"
	"log"
	"net"

	"github.com/novozhenin/practic/internal/transport"
	pb "github.com/novozhenin/practic/pkg/pb"

	"google.golang.org/grpc"
)

// Publisher — gRPC-реализация transport.Publisher.
// Запускает gRPC-сервер, к которому подключаются slave-устройства.
// Команды передаются через канал — паттерн channel-based aggregation.
type Publisher struct {
	pb.UnimplementedServoControlServer
	commands chan transport.Command
	server   *grpc.Server
}

// NewPublisher создаёт и запускает gRPC-сервер на указанном адресе.
func NewPublisher(addr string) (*Publisher, error) {
	p := &Publisher{
		commands: make(chan transport.Command, 16),
	}

	p.server = grpc.NewServer()
	pb.RegisterServoControlServer(p.server, p)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		log.Printf("[master/transport] сервер запущен на %s", addr)
		if err := p.server.Serve(lis); err != nil {
			log.Printf("[master/transport] сервер: %v", err)
		}
	}()

	return p, nil
}

// Publish отправляет команду подключённому slave через канал.
func (p *Publisher) Publish(cmd transport.Command) {
	p.commands <- cmd
}

// Stop останавливает gRPC-сервер.
func (p *Publisher) Stop() {
	p.server.GracefulStop()
}

// CommandStream — реализация bidirectional gRPC stream.
// Slave подключается, master отправляет команды из канала, slave подтверждает.
func (p *Publisher) CommandStream(stream pb.ServoControl_CommandStreamServer) error {
	log.Println("[master/transport] slave подключился")

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// Горутина чтения подтверждений от slave
	go func() {
		defer cancel()
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Printf("[master/transport] slave отключился: %v", err)
				return
			}
			log.Printf("[master/transport] подтверждение: %v", msg.GetAcknowledged())
		}
	}()

	// Основной цикл: чтение из канала команд и отправка в stream
	for {
		select {
		case cmd := <-p.commands:
			direction := "вниз"
			if cmd.DirectionUp {
				direction = "вверх"
			}
			log.Printf("[master/transport] отправка команды: %s", direction)

			if err := stream.Send(&pb.MasterMessage{
				DirectionUp: cmd.DirectionUp,
			}); err != nil {
				return err
			}
		case <-ctx.Done():
			log.Println("[master/transport] stream завершён")
			return ctx.Err()
		}
	}
}
