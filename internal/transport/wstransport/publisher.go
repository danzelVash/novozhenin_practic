package wstransport

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/novozhenin/practic/internal/transport"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Publisher — WebSocket-реализация transport.Publisher.
// Запускает HTTP-сервер, обновляет подключения до WS на /ws.
type Publisher struct {
	commands chan transport.Command
	server   *http.Server
	mu       sync.Mutex
	conn     *websocket.Conn
}

// NewPublisher создаёт и запускает WS-сервер на указанном адресе.
func NewPublisher(addr string) (*Publisher, error) {
	p := &Publisher{
		commands: make(chan transport.Command, 16),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", p.handleUpgrade)

	p.server = &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("[master/transport] WS сервер запущен на %s", addr)
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[master/transport] WS сервер: %v", err)
		}
	}()

	return p, nil
}

// Publish отправляет команду подключённому slave.
func (p *Publisher) Publish(cmd transport.Command) {
	p.commands <- cmd
}

// Stop останавливает HTTP-сервер.
func (p *Publisher) Stop() {
	p.server.Shutdown(context.Background())
}

func (p *Publisher) handleUpgrade(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[master/transport] WS upgrade: %v", err)
		return
	}

	log.Println("[master/transport] slave подключился (WS)")

	p.mu.Lock()
	p.conn = conn
	p.mu.Unlock()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Горутина чтения ack / детекта disconnect
	go func() {
		defer cancel()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[master/transport] slave отключился (WS): %v", err)
				return
			}
		}
	}()

	// Основной цикл: чтение из канала и отправка JSON
	for {
		select {
		case cmd := <-p.commands:
			direction := "вниз"
			if cmd.DirectionUp {
				direction = "вверх"
			}
			log.Printf("[master/transport] отправка команды (WS): %s", direction)

			data, _ := json.Marshal(wsMessage{DirectionUp: cmd.DirectionUp})
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("[master/transport] ошибка отправки (WS): %v", err)
				return
			}
		case <-ctx.Done():
			log.Println("[master/transport] WS stream завершён")
			return
		}
	}
}
