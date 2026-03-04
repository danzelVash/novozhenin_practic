package mqtttransport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/novozhenin/practic/internal/transport"
)

// Subscriber — MQTT-реализация transport.Subscriber.
// Подключается к MQTT-брокеру и слушает топик servo/command.
type Subscriber struct {
	broker string
}

// NewSubscriber создаёт MQTT-подписчика.
func NewSubscriber(broker string) *Subscriber {
	return &Subscriber{broker: broker}
}

// Listen подключается к MQTT-брокеру, подписывается на топик и вызывает handler.
// Блокируется до отмены контекста.
func (s *Subscriber) Listen(ctx context.Context, handler func(transport.Command)) error {
	opts := mqtt.NewClientOptions().
		AddBroker(s.broker).
		SetClientID("slave").
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("подключение к MQTT брокеру: %w", token.Error())
	}
	defer client.Disconnect(250)

	log.Printf("[slave/transport] подключён к MQTT брокеру %s", s.broker)

	token := client.Subscribe(topic, 1, func(_ mqtt.Client, msg mqtt.Message) {
		var m mqttMessage
		if err := json.Unmarshal(msg.Payload(), &m); err != nil {
			log.Printf("[slave/transport] ошибка JSON (MQTT): %v", err)
			return
		}

		cmd := transport.Command{DirectionUp: m.DirectionUp}

		direction := "вниз"
		if cmd.DirectionUp {
			direction = "вверх"
		}
		log.Printf("[slave/transport] получена команда (MQTT): %s", direction)

		handler(cmd)
	})
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("подписка на топик (MQTT): %w", token.Error())
	}

	log.Println("[slave/transport] подписан на топик servo/command")

	<-ctx.Done()
	return ctx.Err()
}
