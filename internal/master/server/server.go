package server

import (
	"log"

	pb "github.com/novozhenin/practic/pkg/pb"
)

// Server — gRPC-сервер для управления slave-устройством через bidirectional stream.
type Server struct {
	pb.UnimplementedServoControlServer
	commands chan bool // канал команд: true=вверх, false=вниз
}

// New создаёт сервер управления сервоприводом.
func New() *Server {
	return &Server{
		commands: make(chan bool, 16),
	}
}

// SendCommand отправляет команду в канал (вызывается из основного конвейера).
func (s *Server) SendCommand(directionUp bool) {
	s.commands <- directionUp
}

// CommandStream — реализация bidirectional stream.
// Slave подключается, master отправляет команды, slave подтверждает.
func (s *Server) CommandStream(stream pb.ServoControl_CommandStreamServer) error {
	log.Println("[server] slave подключился")

	// Горутина для чтения подтверждений от slave
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				log.Printf("[server] slave отключился: %v", err)
				return
			}
			log.Printf("[server] подтверждение от slave: %v", msg.GetAcknowledged())
		}
	}()

	// Основной цикл: отправка команд slave
	for {
		select {
		case cmd := <-s.commands:
			direction := "вниз"
			if cmd {
				direction = "вверх"
			}
			log.Printf("[server] отправка команды slave: %s", direction)

			if err := stream.Send(&pb.MasterMessage{
				DirectionUp: cmd,
			}); err != nil {
				log.Printf("[server] ошибка отправки: %v", err)
				return err
			}
		case <-stream.Context().Done():
			log.Println("[server] stream завершён")
			return stream.Context().Err()
		}
	}
}
