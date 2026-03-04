package client

import (
	"context"
	"fmt"
	"log"

	pb "github.com/novozhenin/practic/pkg/pb"
	"google.golang.org/grpc"
)

// CommandHandler — обработчик полученных команд.
type CommandHandler func(directionUp bool)

// Client — gRPC-клиент для подключения к master.
type Client struct {
	conn    *grpc.ClientConn
	handler CommandHandler
}

// New создаёт клиент управления.
func New(conn *grpc.ClientConn, handler CommandHandler) *Client {
	return &Client{
		conn:    conn,
		handler: handler,
	}
}

// Run подключается к master и слушает команды.
func (c *Client) Run(ctx context.Context) error {
	client := pb.NewServoControlClient(c.conn)

	stream, err := client.CommandStream(ctx)
	if err != nil {
		return fmt.Errorf("открытие stream: %w", err)
	}

	log.Println("[client] подключён к master")

	for {
		msg, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("получение команды: %w", err)
		}

		direction := "вниз"
		if msg.GetDirectionUp() {
			direction = "вверх"
		}
		log.Printf("[client] получена команда: %s", direction)

		c.handler(msg.GetDirectionUp())

		if err := stream.Send(&pb.SlaveMessage{
			Acknowledged: true,
		}); err != nil {
			return fmt.Errorf("отправка подтверждения: %w", err)
		}
	}
}
