package grpctransport

import (
	"context"
	"fmt"
	"log"

	"github.com/novozhenin/practic/internal/transport"
	pb "github.com/novozhenin/practic/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Subscriber — gRPC-реализация transport.Subscriber.
// Подключается к master как gRPC-клиент и слушает команды через bidirectional stream.
type Subscriber struct {
	addr string
}

// NewSubscriber создаёт gRPC-подписчика для подключения к master.
func NewSubscriber(addr string) *Subscriber {
	return &Subscriber{addr: addr}
}

// Listen подключается к master и вызывает handler для каждой полученной команды.
// Блокируется до разрыва соединения или отмены контекста.
func (s *Subscriber) Listen(ctx context.Context, handler func(transport.Command)) error {
	conn, err := grpc.NewClient(s.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("подключение к master: %w", err)
	}
	defer conn.Close()

	client := pb.NewServoControlClient(conn)

	stream, err := client.CommandStream(ctx)
	if err != nil {
		return fmt.Errorf("открытие stream: %w", err)
	}

	log.Println("[slave/transport] подключён к master")

	for {
		msg, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("получение команды: %w", err)
		}

		cmd := transport.Command{DirectionUp: msg.GetDirectionUp()}

		direction := "вниз"
		if cmd.DirectionUp {
			direction = "вверх"
		}
		log.Printf("[slave/transport] получена команда: %s", direction)

		handler(cmd)

		if err := stream.Send(&pb.SlaveMessage{Acknowledged: true}); err != nil {
			return fmt.Errorf("отправка подтверждения: %w", err)
		}
		log.Printf("[slave/transport] подтверждение отправлено master: %s", direction)
	}
}
