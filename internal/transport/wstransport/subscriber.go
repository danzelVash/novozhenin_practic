package wstransport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
	"github.com/novozhenin/practic/internal/transport"
)

// Subscriber — WebSocket-реализация transport.Subscriber.
// Подключается к master как WS-клиент и слушает команды.
type Subscriber struct {
	url string
}

// NewSubscriber создаёт WS-подписчика для подключения к master.
func NewSubscriber(url string) *Subscriber {
	return &Subscriber{url: url}
}

// Listen подключается к master по WS и вызывает handler для каждой полученной команды.
// Блокируется до разрыва соединения или отмены контекста.
func (s *Subscriber) Listen(ctx context.Context, handler func(transport.Command)) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, s.url, nil)
	if err != nil {
		return fmt.Errorf("подключение к master (WS): %w", err)
	}
	defer conn.Close()

	log.Println("[slave/transport] подключён к master (WS)")

	// Горутина для закрытия соединения при отмене контекста
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("получение команды (WS): %w", err)
		}

		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("[slave/transport] ошибка JSON (WS): %v", err)
			continue
		}

		cmd := transport.Command{DirectionUp: msg.DirectionUp}

		direction := "вниз"
		if cmd.DirectionUp {
			direction = "вверх"
		}
		log.Printf("[slave/transport] получена команда (WS): %s", direction)

		handler(cmd)

		// Отправляем ack
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"ack":true}`)); err != nil {
			return fmt.Errorf("отправка подтверждения (WS): %w", err)
		}
		log.Printf("[slave/transport] подтверждение отправлено master (WS): %s", direction)
	}
}
