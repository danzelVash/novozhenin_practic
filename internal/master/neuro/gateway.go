package neuro

import (
	"context"
	"fmt"
	"log"

	pb "github.com/novozhenin/practic/pkg/pb"
	"google.golang.org/grpc"
)

// Gateway — gRPC-клиент к сервису распознавания аудиокоманд.
type Gateway struct {
	client pb.AudioRecognizerClient
}

// NewGateway создаёт шлюз к нейросервису.
func NewGateway(conn grpc.ClientConnInterface) *Gateway {
	return &Gateway{
		client: pb.NewAudioRecognizerClient(conn),
	}
}

// Recognize отправляет аудиофрагмент на распознавание и возвращает команду.
func (g *Gateway) Recognize(ctx context.Context, audio []byte) (string, error) {
	log.Printf("[master/neuro] отправка %d байт на распознавание", len(audio))

	resp, err := g.client.GetAudio(ctx, &pb.GetAudioRequest{
		Chunk: audio,
		Commands: []*pb.GetAudioRequest_Command{
			{Name: "вверх"},
			{Name: "вниз"},
			{Name: ""},
		},
	})
	if err != nil {
		return "", fmt.Errorf("neuro GetAudio: %w", err)
	}

	log.Printf("[master/neuro] распознана команда: %s", resp.GetCommand())
	return resp.GetCommand(), nil
}
