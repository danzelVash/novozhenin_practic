package mqtttransport

import (
	"encoding/json"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/novozhenin/practic/internal/transport"
)

const topic = "servo/command"

// Publisher — MQTT-реализация transport.Publisher.
// Подключается к MQTT-брокеру и публикует команды в топик servo/command.
type Publisher struct {
	client mqtt.Client
}

// NewPublisher создаёт MQTT-паблишера и подключается к брокеру.
func NewPublisher(broker string) (*Publisher, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID("master").
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("подключение к MQTT брокеру: %w", token.Error())
	}

	log.Printf("[master/transport] подключён к MQTT брокеру %s", broker)

	return &Publisher{client: client}, nil
}

// Publish отправляет команду в MQTT топик.
func (p *Publisher) Publish(cmd transport.Command) {
	direction := "вниз"
	if cmd.DirectionUp {
		direction = "вверх"
	}
	log.Printf("[master/transport] отправка команды (MQTT): %s", direction)

	data, _ := json.Marshal(mqttMessage{DirectionUp: cmd.DirectionUp})
	token := p.client.Publish(topic, 1, false, data)
	token.Wait()
	if token.Error() != nil {
		log.Printf("[master/transport] ошибка отправки (MQTT): %v", token.Error())
	}
}

// Stop отключается от MQTT-брокера.
func (p *Publisher) Stop() {
	p.client.Disconnect(250)
}
